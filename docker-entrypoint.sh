#!/bin/sh

USER=abc

echo "---Setup Timezone to ${TZ}---"
echo "${TZ}" > /etc/timezone
echo "---Checking if UID: ${UID} matches user---"
usermod -o -u ${UID} ${USER}
echo "---Checking if GID: ${GID} matches user---"
groupmod -o -g ${GID} ${USER} > /dev/null 2>&1 ||:
usermod -g ${GID} ${USER}
echo "---Setting umask to ${UMASK}---"
umask ${UMASK}

echo "---Taking ownership of data...---"
chown -R ${UID}:${GID} /app /data
chmod +x /app/gh-proxy

echo "make config"
cp /app/config.json.dist /app/config.json
sed -i "s|%WHITE_LIST%|$WHITE_LIST|g" /app/config.json
sed -i "s|%BLACK_LIST%|$BLACK_LIST|g" /app/config.json
sed -i "s|%ALLOW_PROXY_ALL%|$ALLOW_PROXY_ALL|g" /app/config.json
sed -i "s|%OTHER_WHITE_LIST%|$OTHER_WHITE_LIST|g" /app/config.json
sed -i "s|%OTHER_BLACK_LIST%|$OTHER_BLACK_LIST|g" /app/config.json
sed -i "s|%HTTP_HOST%|$HTTP_HOST|g" /app/config.json
sed -i "s|%HTTP_PORT%|$HTTP_PORT|g" /app/config.json
sed -i "s|%SIZE_LIMIT%|$SIZE_LIMIT|g" /app/config.json

echo "Starting..."
su-exec ${USER} /app/gh-proxy "$@"