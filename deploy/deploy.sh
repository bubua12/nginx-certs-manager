#!/bin/bash
# 一键部署脚本 - 在 Ubuntu 服务器上运行
# 用法: bash deploy.sh

set -e

APP_NAME="nginx-certs-manager"
APP_DIR="/opt/$APP_NAME"
WEB_DIR="/var/www/certs-your-domain-com"
SERVICE_FILE="/etc/systemd/system/$APP_NAME.service"

echo "=== Nginx Certs Manager 部署 ==="

# 创建目录
echo "1. 创建目录..."
mkdir -p $APP_DIR/data
mkdir -p $WEB_DIR

# 检查二进制文件
if [ ! -f "./$APP_NAME" ]; then
    echo "错误: 找不到 $APP_NAME 二进制文件"
    echo "请先在开发机执行 make build-linux，然后将 build/ 下的文件传到服务器"
    exit 1
fi

# 复制后端二进制
echo "2. 部署后端..."
cp ./$APP_NAME $APP_DIR/$APP_NAME
chmod +x $APP_DIR/$APP_NAME

# 复制前端静态文件
echo "3. 部署前端..."
if [ -d "./web-dist" ]; then
    rm -rf $WEB_DIR/*
    cp -r ./web-dist/* $WEB_DIR/
elif [ -d "./dist" ]; then
    rm -rf $WEB_DIR/*
    cp -r ./dist/* $WEB_DIR/
else
    echo "警告: 找不到前端文件，跳过前端部署"
fi

# 安装 systemd 服务
echo "4. 配置 systemd 服务..."
cat > $SERVICE_FILE << EOF
[Unit]
Description=Nginx Certs Manager
After=network.target nginx.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/nginx-certs-manager
ExecStart=/opt/nginx-certs-manager/nginx-certs-manager
Restart=on-failure
RestartSec=5
Environment=PORT=8080
Environment=DB_PATH=/opt/nginx-certs-manager/data/certs.db
Environment=WEB_DIR=$WEB_DIR
Environment=NGINX_DIR=/etc/nginx
Environment=CERTBOT_DIR=/etc/letsencrypt
Environment=JWT_SECRET=$(openssl rand -hex 32)
Environment=ADMIN_PASSWORD=$(openssl rand -base64 16)

[Install]
WantedBy=multi-user.target
EOF

# 启动服务
echo "5. 启动服务..."
systemctl daemon-reload
systemctl enable $APP_NAME
systemctl restart $APP_NAME

echo ""
echo "=== 部署完成 ==="
echo "服务状态: systemctl status $APP_NAME"
echo "查看日志: journalctl -u $APP_NAME -f"
echo ""
echo "请自行配置 Nginx 反向代理 + SSL 证书，然后访问你的域名"
