## Nginx Certs Manager

管理 Let's Encrypt + Certbot + Nginx SSL 证书的可视化工具。

### 快速开始

#### 开发模式

```bash
# 终端 1: 启动后端 (port 8080)
make dev-backend

# 终端 2: 启动前端 (port 3000, 自动代理 API 到 8080)
make dev-frontend
```

浏览器访问 http://localhost:3000

#### 构建 & 部署

```bash
# 构建前后端
make build

# 安装到 /opt/nginx-certs-manager
make install

# 配置 systemd 服务
sudo cp deploy/nginx-certs-manager.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable nginx-certs-manager
sudo systemctl start nginx-certs-manager
```

浏览器访问 http://your-server:8080

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| PORT | 8080 | 服务端口 |
| DB_PATH | ./data/certs.db | SQLite 数据库路径 |
| WEB_DIR | ./web/dist | 前端静态文件目录 |
| NGINX_DIR | /etc/nginx | Nginx 配置目录 |
| CERTBOT_DIR | /etc/letsencrypt | Certbot 证书目录 |

### 功能

- 仪表盘：证书统计、到期时间线、Nginx 状态
- 证书管理：列表、详情、续期、申请、撤销
- 站点管理：列表、启用/禁用、配置编辑、配置校验
- 系统设置：扫描间隔、通知配置
- 操作日志：所有操作记录
