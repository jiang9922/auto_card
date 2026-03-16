# Autocard 项目上线前系统检查报告

**检查时间**: 2026-03-17  
**检查人员**: 系统工程师  
**项目路径**: `/root/.openclaw/workspace/auto_card/`

---

## 1. 项目配置检查

### 1.1 环境变量配置

| 变量名 | 用途 | 默认值 | 状态 |
|--------|------|--------|------|
| `PORT` | 服务监听端口 | 8080 | ✅ 正确配置 |
| `RAILWAY_PUBLIC_DOMAIN` | Railway 公共域名 | - | ✅ 自动注入 |
| `VITE_API_BASE_URL` | 前端 API 基础地址 | '' (空) | ⚠️ 生产环境建议配置 |

### 1.2 发现的问题

| 问题 | 严重程度 | 说明 |
|------|----------|------|
| 管理员密码硬编码 | 🔴 **高风险** | `main.go:116` 硬编码密码 `jc123` |
| 数据库路径写死 | 🟡 中风险 | `./cards.db` 路径固定，容器内数据可能丢失 |
| 缺少 .env.production | 🟡 中风险 | 前端缺少生产环境配置 |
| API 路径依赖相对路径 | 🟡 中风险 | `getBaseURL()` 返回空字符串时使用相对路径 |

---

## 2. Docker 配置检查

### 2.1 Dockerfile 分析

```dockerfile
# 多阶段构建结构
阶段1: frontend-builder (node:20-alpine) - 构建前端
阶段2: backend-builder (golang:1.22-alpine) - 构建后端  
阶段3: 最终镜像 (alpine:latest) - 运行环境
```

### 2.2 Dockerfile 检查清单

| 检查项 | 状态 | 备注 |
|--------|------|------|
| 多阶段构建 | ✅ | 正确使用了多阶段构建 |
| 前端 Node 版本 | ✅ | node:20-alpine |
| 后端 Go 版本 | ✅ | golang:1.22-alpine |
| 基础镜像 | ⚠️ | alpine:latest 建议使用固定版本如 alpine:3.19 |
| CGO 启用 | ✅ | CGO_ENABLED=1 用于 SQLite |
| 依赖安装 | ✅ | gcc musl-dev sqlite-dev |
| 暴露端口 | ✅ | EXPOSE 8080 |
| 数据卷 | ❌ | 未定义 VOLUME 挂载点 |

### 2.3 发现的问题

1. **数据持久化问题**: SQLite 数据库文件存储在容器内 `/root/cards.db`，容器重启后数据会丢失
2. **缺少健康检查**: 未配置 HEALTHCHECK
3. **时区问题**: 未设置时区，可能影响时间记录

---

## 3. 部署配置检查

### 3.1 Railway 配置 (railway.toml)

```toml
[build]
builder = "DOCKERFILE"
dockerfilePath = "Dockerfile"
```

| 检查项 | 状态 | 备注 |
|--------|------|------|
| 构建器配置 | ✅ | 使用 Dockerfile |
| 部署配置 | ❌ | 缺少 `[deploy]` 区块 |
| 健康检查 | ❌ | 未配置健康检查 |
| 环境变量 | ❌ | 未定义所需环境变量 |

### 3.2 Vercel 配置 (vercel.json)

```json
{
  "rewrites": [
    { "source": "/api/(.*)", "destination": "/api/index.go" },
    { "source": "/(.*)", "destination": "/index.html" }
  ]
}
```

| 检查项 | 状态 | 备注 |
|--------|------|------|
| API 路由 | ⚠️ | 指向 `/api/index.go` 但该文件不存在 |
| 前端路由 | ✅ | SPA 回退配置正确 |

⚠️ **警告**: Vercel 配置与项目实际结构不匹配，项目使用 Go 后端而非 Serverless Function

---

## 4. 数据库迁移检查

### 4.1 数据库初始化

**位置**: `backend/main.go:31-53`

```go
// 数据库表结构自动创建
CREATE TABLE IF NOT EXISTS cards (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    card_no TEXT NOT NULL,
    card_link TEXT NOT NULL,
    phone TEXT,
    remark TEXT,
    query_url TEXT,
    query_token TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    card_code TEXT,
    card_expired_date TEXT,
    card_note TEXT,
    card_check BOOLEAN DEFAULT FALSE
);
```

### 4.2 检查结论

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 迁移脚本 | ⚠️ | 使用 `init()` 自动建表，无独立迁移脚本 |
| 版本控制 | ❌ | 无数据库版本控制机制 |
| 回滚机制 | ❌ | 无回滚方案 |

### 4.3 建议

1. 使用独立的数据库迁移工具（如 golang-migrate）
2. 建立版本控制表 `schema_migrations`
3. 编写增量迁移脚本而非每次全量重建

---

## 5. 上线部署文档

详见 [DEPLOY.md](./DEPLOY.md)

---

## 6. 紧急修复建议（上线前必须完成）

### 6.1 立即修复（阻塞上线）

1. **修复硬编码密码**
   ```go
   // 修改前
   if req.Password != "jc123" { ... }
   
   // 修改后
   adminPassword := os.Getenv("ADMIN_PASSWORD")
   if adminPassword == "" {
       adminPassword = "jc123" // 仅开发环境默认值
   }
   if req.Password != adminPassword { ... }
   ```

2. **配置数据持久化**
   ```dockerfile
   # Dockerfile 添加
   VOLUME ["/data"]
   ENV DB_PATH=/data/cards.db
   ```

3. **更新数据库初始化**
   ```go
   dbPath := os.Getenv("DB_PATH")
   if dbPath == "" {
       dbPath = "./cards.db"
   }
   db, err = sql.Open("sqlite3", dbPath)
   ```

### 6.2 强烈建议（上线前完成）

1. 添加 Dockerfile HEALTHCHECK
2. 配置 Railway 环境变量
3. 移除或修复 Vercel 配置
4. 添加前端生产环境配置 `.env.production`

---

## 7. 检查结论

| 项目 | 状态 |
|------|------|
| 项目配置 | 🟡 **需要修改** |
| Docker 配置 | 🟡 **需要优化** |
| 部署配置 | 🔴 **需要修复** |
| 数据库迁移 | 🟡 **建议改进** |
| 整体状态 | 🔴 **不建议立即上线** |

**建议**: 完成 6.1 节的修复后再进行上线部署。

---

*报告生成时间: 2026-03-17*
