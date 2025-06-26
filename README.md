# gh-proxy-go

这是 [gh-proxy](https://github.com/hunshcn/gh-proxy) 的 Go 语言版本。

> ### **修改说明 (Modification Note)**
>
> 本项目是 [moeyy01/gh-proxy-go](https://github.com/moeyy01/gh-proxy-go) 的一个修改版本。

## 使用方法

```bash
docker run -d \
  --name gh-proxy \
  --restart=unless-stopped \
  -p 8080:8080 \
  -v /data:/config \
  -e "TZ=Asia/Shanghai" \
  ghcr.io/t0saki/gh-proxy-go:main
```

## 部署详情

以下是原项目 `github.moeyy.xyz` 实例的部署和性能详情，可供参考：

[github.moeyy.xyz](https://github.moeyy.xyz/) 正在使用 **gh-proxy-go**，托管在 [BuyVM](https://buyvm.net/) 每月 3.5 美元的 1 核 1G 内存、10Gbps 带宽服务器上。

### 服务器概况：

- **日流量处理**：约 3TB
- **CPU 平均使用率**：20%
- **带宽平均占用**：400Mbps

![服务器数据](https://github.com/user-attachments/assets/6fe37f41-aa35-4efc-b0b8-8c3339529326)
![Cloudflare 数据](https://github.com/user-attachments/assets/ae310b1f-96e9-42e9-a77c-0d8c1b8d6344)

---

如有问题或改进建议，欢迎提交 issue 或 PR！
