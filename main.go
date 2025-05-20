package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/fs"
	"net"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	sizeLimit int64 = 1024 * 1024 * 1024 * 10 // 允许的文件大小，默认10GB
	host            = "0.0.0.0"               // 监听地址
	port            = 8080                    // 监听端口
)

//go:embed public/*
var public embed.FS

var (
	exps = []*regexp.Regexp{
		regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:releases|archive)/.*$`),
		regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:blob|raw)/.*$`),
		regexp.MustCompile(`^(?:https?://)?github\.com/([^/]+)/([^/]+)/(?:info|git-).*$`),
		regexp.MustCompile(`^(?:https?://)?raw\.github(?:usercontent|)\.com/([^/]+)/([^/]+)/.+?/.+$`),
		regexp.MustCompile(`^(?:https?://)?gist\.github\.com/([^/]+)/.+?/.+$`),
	}
	httpClient *http.Client
	config     *Config
	configLock sync.RWMutex
)

type Config struct {
	Host           string   `json:"host"`
	Port           int64    `json:"port"`
	SizeLimit      int64    `json:"sizeLimit"`
	WhiteList      []string `json:"whiteList"`
	BlackList      []string `json:"blackList"`
	AllowProxyAll  bool     `json:"allowProxyAll"` // 是否允许代理非github的其他地址
	OtherWhiteList []string `json:"otherWhiteList"`
	OtherBlackList []string `json:"otherBlackList"`
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	httpClient = &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          1000,
			MaxIdleConnsPerHost:   1000,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ResponseHeaderTimeout: 300 * time.Second,
		},
	}

	loadConfig()
	go func() {
		for {
			time.Sleep(10 * time.Minute)
			loadConfig()
		}
	}()
	// if config.Host==""{
	// 	config.Host=host
	// }
	if config.Port == 0 {
		config.Port = port
	}
	if config.SizeLimit <= 0 {
		config.SizeLimit = sizeLimit
	}
	// 修改静态文件服务方式
	subFS, err := fs.Sub(public, "public")
	if err != nil {
		panic(fmt.Sprintf("无法创建子文件系统: %v", err))
	}

	// 使用 StaticFS 提供静态文件
	router.StaticFS("/", http.FS(subFS))
	// router.StaticFile("/", "./public/index.html")
	// router.StaticFile("/favicon.ico", "./public/favicon.ico")
	// router.StaticFile("/logo.png", "./public/logo.png")
	router.NoRoute(handler)
	fmt.Printf("starting http server on %s:%d \n", config.Host, config.Port)
	err = router.Run(fmt.Sprintf("%s:%d", config.Host, config.Port))
	if err != nil {
		fmt.Printf("Error starting server: %v\n", err)
	}
}

func handler(c *gin.Context) {
	rawPath := strings.TrimPrefix(c.Request.URL.RequestURI(), "/")

	for strings.HasPrefix(rawPath, "/") {
		rawPath = strings.TrimPrefix(rawPath, "/")
	}

	if !strings.HasPrefix(rawPath, "http") {
		// c.String(http.StatusForbidden, "Invalid input.")
		// return
		rawPath = fmt.Sprintf("https://%s", rawPath)
	}

	matches := checkURL(rawPath)
	if matches != nil {
		if len(config.WhiteList) > 0 && !checkList(matches, config.WhiteList) {
			c.String(http.StatusForbidden, "Forbidden by white list.")
			return
		}
		if len(config.BlackList) > 0 && checkList(matches, config.BlackList) {
			c.String(http.StatusForbidden, "Forbidden by black list.")
			return
		}
	} else {
		if !config.AllowProxyAll {
			c.String(http.StatusForbidden, "Invalid input.")
			return
		}
		if len(config.OtherWhiteList) > 0 && !checkOhterList(rawPath, config.OtherWhiteList) {
			c.String(http.StatusForbidden, "Forbidden by white list.")
			return
		}
		if len(config.OtherBlackList) > 0 && checkOhterList(rawPath, config.OtherBlackList) {
			c.String(http.StatusForbidden, "Forbidden by black list.")
			return
		}
	}

	if exps[1].MatchString(rawPath) {
		rawPath = strings.Replace(rawPath, "/blob/", "/raw/", 1)
	}

	proxy(c, rawPath)
}

func proxy(c *gin.Context, u string) {
	req, err := http.NewRequest(c.Request.Method, u, c.Request.Body)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
		return
	}

	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}
	req.Header.Del("Host")

	resp, err := httpClient.Do(req)
	if err != nil {
		c.String(http.StatusInternalServerError, fmt.Sprintf("server error %v", err))
		return
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {

		}
	}(resp.Body)

	if contentLength, ok := resp.Header["Content-Length"]; ok {
		if size, err := strconv.ParseInt(contentLength[0], 10, 64); err == nil && size > config.SizeLimit {
			c.String(http.StatusRequestEntityTooLarge, "File too large.")
			return
		}
	}

	resp.Header.Del("Content-Security-Policy")
	resp.Header.Del("Referrer-Policy")
	resp.Header.Del("Strict-Transport-Security")

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	if location := resp.Header.Get("Location"); location != "" {
		if checkURL(location) != nil {
			c.Header("Location", "/"+location)
		} else {
			proxy(c, location)
			return
		}
	}

	c.Status(resp.StatusCode)
	if _, err := io.Copy(c.Writer, resp.Body); err != nil {
		return
	}
}

func loadConfig() {
	file, err := os.Open("config.json")
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {

		}
	}(file)

	var newConfig Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&newConfig); err != nil {
		fmt.Printf("Error decoding config: %v\n", err)
		return
	}

	configLock.Lock()
	config = &newConfig
	configLock.Unlock()
}

func checkURL(u string) []string {
	for _, exp := range exps {
		if matches := exp.FindStringSubmatch(u); matches != nil {
			return matches[1:]
		}
	}
	return nil
}

func checkList(matches, list []string) bool {
	for _, item := range list {
		if strings.HasPrefix(matches[0], item) {
			return true
		}
	}
	return false
}

func checkOhterList(url string, list []string) bool {
	for _, item := range list {
		if strings.Contains(url, item) {
			return true
		}
	}
	return false
}
