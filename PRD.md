# Nginx Certs Manager — PRD

## Context

用户有一台 Ubuntu 公网服务器，使用 Let's Encrypt + Certbot + Nginx 管理 HTTPS 证书。由于配置了大量子域名，证书管理变得分散且难以追踪。需要一个可视化管理面板，集中管理所有证书和 Nginx 站点配置。

## 技术栈

| 层 | 技术 |
|---|------|
| 后端 | Go + Echo (轻量高性能) |
| 前端 | Vue 3 + Vite + Element Plus |
| 数据库 | SQLite (go-sqlite3) |
| 部署 | 单二进制 + 内嵌前端静态文件 |

## 整体架构

```
┌─────────────────────────────────────────────┐
│                  浏览器 UI                    │
│         Vue 3 + Element Plus SPA             │
└─────────────────┬───────────────────────────┘
                  │ HTTP API (JSON)
┌─────────────────▼───────────────────────────┐
│             Go Backend (Echo)                │
│  ┌──────────┐ ┌──────────┐ ┌──────────────┐ │
│  │ 证书管理  │ │ 站点管理  │ │  系统监控     │ │
│  └────┬─────┘ └────┬─────┘ └──────┬───────┘ │
│       │            │              │          │
│  ┌────▼────────────▼──────────────▼───────┐  │
│  │          SQLite (数据持久化)            │  │
│  └────────────────────────────────────────┘  │
│       │            │              │          │
│  ┌────▼─────┐ ┌────▼─────┐ ┌─────▼──────┐   │
│  │ Certbot  │ │  Nginx   │ │  系统命令   │   │
│  │  CLI     │ │  CLI     │ │  (cron等)   │   │
│  └──────────┘ └──────────┘ └────────────┘   │
└─────────────────────────────────────────────┘
```

## 功能模块

### P0 — 第一期 (MVP)

#### 1. 仪表盘 (Dashboard)

- 证书总数、即将过期数量、已过期数量（卡片展示）
- 证书到期时间线（甘特图/时间轴，按颜色区分：绿色=健康，黄色=<30天，红色=已过期/即将过期）
- 站点总数、活跃站点数
- 最近续期操作日志

#### 2. 证书管理 (Certificates)

- **证书列表**：域名、到期时间、剩余天数、状态标签、签发机构、操作按钮
- **证书详情**：查看完整证书信息（CN、SAN、指纹、有效期、密钥类型等）
- **手动续期**：一键触发 `certbot renew --cert-name <domain>`
- **自动续期状态**：显示 cron/systemd timer 状态，支持开关
- **续期历史**：每次续期操作的结果和日志
- **证书申请**：通过 UI 发起 `certbot certonly` 申请新证书
- **证书撤销**：`certbot revoke` 危险操作需二次确认

#### 3. Nginx 站点管理 (Sites)

- **站点列表**：域名、端口、SSL 状态、启用/禁用状态
- **站点详情**：查看完整 server block 配置
- **启用/禁用站点**：`ln -s` / `rm` sites-enabled 的软链接 + `nginx -s reload`
- **配置校验**：`nginx -t` 检测配置是否正确
- **Nginx 状态**：运行状态、版本、PID

### P1 — 第二期

#### 4. 域名管理 (Domains)

- 域名与证书的关联关系展示
- 域名解析状态检查（DNS A/AAAA 记录 vs 服务器 IP）
- 新域名接入向导（DNS 检查 → 申请证书 → 创建 Nginx 配置 → 完成）

#### 5. Nginx 日志可视化 (Logs)

- 实时日志流（WebSocket tail access.log）
- 请求量统计（按小时/天聚合折线图）
- 状态码分布（饼图）
- Top URL / Top IP 排行
- 响应时间分布

### P2 — 第三期

#### 6. 系统设置

- 通知配置（证书即将过期时发送通知：邮件 / Webhook / Telegram）
- 自动续期时间配置
- 备份与恢复（Nginx 配置文件 + 证书文件）
- 操作审计日志

#### 7. 安全

- 登录认证（用户名密码 + JWT）
- 操作权限分级
- 操作日志审计

## 数据库设计

```sql
-- 证书表
CREATE TABLE certificates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL UNIQUE,
    sans TEXT,                    -- JSON array of SAN domains
    cert_path TEXT,               -- /etc/letsencrypt/live/<domain>/fullchain.pem
    key_path TEXT,                -- /etc/letsencrypt/live/<domain>/privkey.pem
    issuer TEXT,                  -- Let's Encrypt
    not_before DATETIME,
    not_after DATETIME,
    auto_renew BOOLEAN DEFAULT 1,
    status TEXT DEFAULT 'active', -- active, expiring, expired, revoked
    last_renewed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Nginx 站点表
CREATE TABLE sites (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    domain TEXT NOT NULL UNIQUE,
    config_path TEXT,             -- /etc/nginx/sites-available/<domain>
    port INTEGER DEFAULT 443,
    upstream TEXT,                -- 反向代理目标
    ssl_enabled BOOLEAN DEFAULT 1,
    certificate_id INTEGER REFERENCES certificates(id),
    enabled BOOLEAN DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 操作日志表
CREATE TABLE operation_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    type TEXT NOT NULL,           -- cert_renew, site_enable, site_disable, cert_request...
    target TEXT,                  -- 操作对象 (域名等)
    status TEXT,                  -- success, failed
    message TEXT,                 -- 操作结果详情
    operator TEXT DEFAULT 'system',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- 系统配置表
CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
```

## API 设计

```
GET    /api/dashboard/stats              # 仪表盘统计数据
GET    /api/dashboard/timeline            # 证书到期时间线

GET    /api/certificates                  # 证书列表
GET    /api/certificates/:id              # 证书详情
POST   /api/certificates/renew/:id        # 手动续期
POST   /api/certificates/request          # 申请新证书
DELETE /api/certificates/:id              # 撤销证书

GET    /api/sites                         # 站点列表
GET    /api/sites/:id                     # 站点详情
POST   /api/sites/:id/enable              # 启用站点
POST   /api/sites/:id/disable             # 禁用站点
POST   /api/sites/validate                # 校验 Nginx 配置
GET    /api/sites/:id/config              # 获取配置内容
PUT    /api/sites/:id/config              # 更新配置内容

GET    /api/nginx/status                  # Nginx 运行状态
POST   /api/nginx/reload                  # 重新加载 Nginx

GET    /api/logs                          # 操作日志列表
GET    /api/settings                      # 系统设置
PUT    /api/settings                      # 更新设置
```

## 前端页面结构

```
/                           → 重定向到 /dashboard
/dashboard                  → 仪表盘
/certificates               → 证书列表
/certificates/:id           → 证书详情
/sites                      → 站点列表
/sites/:id                  → 站点详情
/settings                   → 系统设置
```

## 项目目录结构

```
nginx-certs-manager/
├── cmd/
│   └── server/
│       └── main.go              # 入口，启动 Echo 服务
├── internal/
│   ├── config/
│   │   └── config.go            # 配置加载
│   ├── database/
│   │   ├── db.go                # SQLite 初始化
│   │   └── migrations.go        # 建表迁移
│   ├── model/
│   │   ├── certificate.go       # 证书模型
│   │   ├── site.go              # 站点模型
│   │   └── log.go               # 日志模型
│   ├── handler/
│   │   ├── dashboard.go         # 仪表盘 handler
│   │   ├── certificate.go       # 证书 handler
│   │   ├── site.go              # 站点 handler
│   │   ├── nginx.go             # Nginx 操作 handler
│   │   └── settings.go          # 设置 handler
│   ├── service/
│   │   ├── certbot.go           # Certbot CLI 封装
│   │   ├── nginx.go             # Nginx CLI 封装
│   │   └── scanner.go           # 证书/站点自动扫描
│   └── middleware/
│       └── logger.go            # 请求日志
├── web/                         # 前端项目
│   ├── src/
│   │   ├── api/                 # API 请求封装
│   │   ├── views/               # 页面组件
│   │   │   ├── Dashboard.vue
│   │   │   ├── certificates/
│   │   │   ├── sites/
│   │   │   └── settings/
│   │   ├── components/          # 公共组件
│   │   ├── router/
│   │   ├── stores/              # Pinia 状态管理
│   │   └── App.vue
│   ├── package.json
│   └── vite.config.ts
├── go.mod
├── go.sum
├── Makefile                     # 构建脚本
└── README.md
```

## 实施计划

### 第一步：后端骨架
- Go module 初始化 + Echo 路由框架
- SQLite 数据库初始化 + 迁移
- 证书自动扫描（读取 /etc/letsencrypt/ 解析证书信息写入数据库）
- Nginx 站点自动扫描（读取 /etc/nginx/sites-enabled/ 写入数据库）

### 第二步：核心 API
- Dashboard 统计 API
- 证书 CRUD + Certbot CLI 封装
- 站点 CRUD + Nginx CLI 封装
- 操作日志

### 第三步：前端开发
- Vue 3 项目初始化 + Element Plus + Vue Router + Pinia
- 仪表盘页面（统计卡片 + 到期时间线）
- 证书管理页面（列表 + 详情 + 续期操作）
- 站点管理页面（列表 + 详情 + 启用/禁用）
- 设置页面

### 第四步：集成 & 部署
- 前端构建产物内嵌到 Go 二进制（go:embed）
- Makefile 一键构建
- 部署脚本（systemd service 配置）

## 验证方式

1. 本地开发：`go run cmd/server/main.go` 启动后端，`npm run dev` 启动前端
2. API 测试：curl 各接口确认返回正确
3. 前端页面：浏览器访问确认 UI 展示和交互正常
4. 证书操作：在测试环境验证 certbot 命令调用
5. 打包部署：`make build` 生成单二进制，systemd 启动后通过浏览器访问
