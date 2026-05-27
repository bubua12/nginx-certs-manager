<h1 align="center">🔐 Nginx Certs Manager</h1>

<p align="center">
  <strong>Let's Encrypt 证书 & Nginx 站点可视化管理面板</strong>
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go">
  <img src="https://img.shields.io/badge/Vue-3.4+-4FC08D?style=flat-square&logo=vue.js&logoColor=white" alt="Vue">
  <img src="https://img.shields.io/badge/Element_Plus-2.7+-409EFF?style=flat-square" alt="Element Plus">
  <img src="https://img.shields.io/badge/SQLite-3-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite">
  <img src="https://img.shields.io/badge/License-MIT-blue?style=flat-square" alt="License">
</p>

---

## ✨ 功能特性

| 模块 | 功能 |
|------|------|
| 📊 **仪表盘** | 证书统计卡片、到期时间线图表、Nginx 运行状态 |
| 🔒 **证书管理** | 查看详情、一键续期、申请新证书、撤销证书 |
| 🌐 **站点管理** | 自动发现 Nginx 站点、查看/编辑配置、启用/禁用站点 |
| ⚙️ **系统设置** | 扫描间隔、通知配置、操作日志查看 |
| 🔑 **登录认证** | JWT 认证、IP 限流锁定（5次失败锁30分钟）、修改密码 |
| 🛡️ **安全防护** | 密码 bcrypt 加密、Token 24h 过期、无公开注册入口 |

## 📁 项目结构

```
nginx-certs-manager/
├── cmd/server/main.go              # 后端入口
├── internal/
│   ├── config/                     # 环境变量配置
│   ├── database/                   # SQLite + 自动迁移 + 默认管理员
│   ├── model/                      # 数据模型 (User, Certificate, Site, Log)
│   ├── handler/                    # API 接口 (Auth, Dashboard, Cert, Site, Nginx)
│   ├── service/                    # 业务逻辑 (Certbot, Nginx, Scanner, IPLockout)
│   └── middleware/                  # JWT 认证中间件
├── web/                            # Vue 3 前端
│   └── src/
│       ├── views/                  # 页面 (Login, Dashboard, Certs, Sites, Settings)
│       ├── stores/                 # Pinia 状态管理 (Auth)
│       ├── api/                    # Axios 封装
│       └── router/                 # 路由 + 登录守卫
├── deploy/                         # 部署配置 (systemd, nginx vhost, 脚本)
├── Makefile                        # 构建脚本
└── PRD.md                          # 产品需求文档
```

## 🚀 快速开始

### 开发模式

```bash
# 终端 1：启动后端 (port 8080)
go run ./cmd/server/

# 终端 2：启动前端 (port 3000, 自动代理 API)
cd web && npm run dev
```

浏览器访问 `http://localhost:3000`

### 生产部署

```bash
# 1. 构建 (在开发机上，交叉编译 Linux)
make build-linux

# 2. 上传到服务器
scp build/nginx-certs-manager root@your-server:/opt/nginx-certs-manager/
scp -r web/dist/* root@your-server:/var/www/certs.your-domain.com/

# 3. 服务器上执行部署脚本
bash deploy.sh
```

## ⚙️ 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `PORT` | `8080` | 服务端口 |
| `DB_PATH` | `./data/certs.db` | SQLite 数据库路径 |
| `WEB_DIR` | `./web/dist` | 前端静态文件目录 |
| `NGINX_DIR` | `/etc/nginx` | Nginx 配置目录 |
| `CERTBOT_DIR` | `/etc/letsencrypt` | Certbot 证书目录 |
| `JWT_SECRET` | `nginx-certs-manager-default-secret` | JWT 签名密钥（生产环境务必修改） |
| `ADMIN_PASSWORD` | `admin` | 默认管理员密码（仅首次启动生效，登录后请立即修改） |

## 🔑 默认账号

| 项目 | 值 |
|------|-----|
| 用户名 | `admin` |
| 密码 | `admin` |

> ⚠️ 首次登录后请立即修改密码！密码可通过环境变量 `ADMIN_PASSWORD` 自定义（需在首次启动前设置）。

## 🌐 Nginx 配置示例

```nginx
server {
    listen 443 ssl;
    server_name certs.your-domain.com;

    ssl_certificate     /etc/letsencrypt/live/certs.your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/certs.your-domain.com/privkey.pem;

    root /var/www/certs.your-domain.com;
    index index.html;

    location / {
        try_files $uri $uri/ /index.html;
    }

    location /api {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

## 🔧 技术栈

**后端：** Go + Echo + GORM + SQLite (纯 Go, 无 CGO 依赖)

**前端：** Vue 3 + Vite + Element Plus + ECharts + Pinia

**证书扫描：**
- 优先直接读取 `/etc/letsencrypt/live/` 下的证书文件 (x509 解析)
- 备选 `certbot certificates` 命令

**站点发现：**
- 自动解析 `/etc/nginx/nginx.conf` 中的 server 块
- 扫描 `/etc/nginx/conf.d/*.conf`
- 兼容传统 `sites-available/` + `sites-enabled/` 布局

## 📝 API 接口

```
POST   /api/auth/login              # 登录
POST   /api/auth/change-password    # 修改密码
GET    /api/auth/me                 # 当前用户信息

GET    /api/dashboard/stats         # 仪表盘统计
GET    /api/dashboard/timeline      # 证书到期时间线

GET    /api/certificates            # 证书列表
GET    /api/certificates/:id        # 证书详情
POST   /api/certificates/renew/:id  # 续期证书
POST   /api/certificates/request    # 申请证书
DELETE /api/certificates/:id        # 撤销证书

GET    /api/sites                   # 站点列表
GET    /api/sites/:id               # 站点详情
GET    /api/sites/:id/config        # 获取配置
PUT    /api/sites/:id/config        # 更新配置
POST   /api/sites/:id/enable        # 启用站点
POST   /api/sites/:id/disable       # 禁用站点

GET    /api/nginx/status            # Nginx 状态
POST   /api/nginx/reload            # 重载 Nginx
POST   /api/nginx/validate          # 校验配置

GET    /api/logs                    # 操作日志
GET    /api/settings                # 系统设置
PUT    /api/settings                # 更新设置
```

## 📄 License

MIT
