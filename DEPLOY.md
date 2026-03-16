# Autocard 项目上线部署文档

## 📋 部署前准备

### 1. 环境要求

| 组件 | 版本要求 |
|------|----------|
| Docker | >= 20.0 |
| Docker Compose | >= 2.0 (可选) |
| Git | 任意版本 |
| Node.js | 20.x (本地开发) |
| Go | 1.22+ (本地开发) |

### 2. 必要的环境变量

在部署前，请配置以下环境变量：

```bash
# 基础配置
PORT=8080                          # 服务端口
ADMIN_PASSWORD=your_secure_password  # 管理员密码（必须修改！）

# 数据库配置
DB_PATH=/data/cards.db             # 数据库文件路径（容器内）

# 前端配置（生产环境）
VITE_API_BASE_URL=                 # 留空使用相对路径，或填写完整 API 地址
```

---

## 🚀 部署方式

### 方式一：Railway 部署（推荐）

#### 步骤 1：准备代码

```bash
git clone <your-repo-url>
cd auto_card
```

#### 步骤 2：创建 Railway 项目

1. 访问 [Railway Dashboard](https://railway.app/dashboard)
2. 点击 "New Project"
3. 选择 "Deploy from GitHub repo"
4. 选择你的仓库

#### 步骤 3：配置环境变量

在 Railway Dashboard 的 Variables 标签页添加：

```
PORT=8080
ADMIN_PASSWORD=your_secure_password_here
DB_PATH=/data/cards.db
```

⚠️ **重要**: `ADMIN_PASSWORD` 必须使用强密码！

#### 步骤 4：配置数据持久化

1. 在 Railway Dashboard 点击 "New" → "Volume"
2. 命名卷为 `data`
3. 挂载路径设置为 `/data`
4. 重新部署服务

#### 步骤 5：部署

Railway 会自动检测 `railway.toml` 并使用 Dockerfile 构建部署。

---

### 方式二：VPS/Dedicated Server 部署

#### 步骤 1：准备服务器

```bash
# 安装 Docker
curl -fsSL https://get.docker.com | sh

# 启动 Docker
systemctl start docker
systemctl enable docker
```

#### 步骤 2：构建镜像

```bash
git clone <your-repo-url>
cd auto_card

# 构建 Docker 镜像
docker build -t autocard:latest .
```

#### 步骤 3：运行容器

```bash
# 创建数据目录
mkdir -p /var/lib/autocard/data

# 运行容器
docker run -d \
  --name autocard \
  -p 8080:8080 \
  -e PORT=8080 \
  -e ADMIN_PASSWORD=your_secure_password \
  -e DB_PATH=/data/cards.db \
  -v /var/lib/autocard/data:/data \
  --restart unless-stopped \
  autocard:latest
```

#### 步骤 4：配置 Nginx 反向代理（可选）

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_cache_bypass $http_upgrade;
    }
}
```

---

### 方式三：Docker Compose 部署

#### docker-compose.yml

```yaml
version: '3.8'

services:
  autocard:
    build: .
    container_name: autocard
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-jc123}
      - DB_PATH=/data/cards.db
      - GIN_MODE=release
    volumes:
      - ./data:/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "-q", "--spider", "http://localhost:8080/api/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

#### 部署命令

```bash
# 启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

---

## 📁 项目结构说明

```
auto_card/
├── backend/
│   ├── main.go          # 后端主程序
│   ├── go.mod           # Go 依赖
│   └── cards.db         # SQLite 数据库（运行后生成）
├── frontend/
│   ├── src/             # 前端源代码
│   ├── dist/            # 构建产物（运行后生成）
│   └── package.json     # Node 依赖
├── api/                 # Vercel Serverless API（当前未使用）
├── mock_sms_api/        # 短信模拟接口
├── Dockerfile           # Docker 构建配置
├── railway.toml         # Railway 部署配置
└── vercel.json          # Vercel 部署配置（当前不匹配）
```

---

## 🔧 配置修改指南

### 修改管理员密码

**方法 1：环境变量（推荐）**

在部署平台设置 `ADMIN_PASSWORD` 环境变量。

**方法 2：修改代码**

编辑 `backend/main.go` 第 116 行：

```go
// 修改前
if req.Password != "jc123" {

// 修改后  
adminPass := os.Getenv("ADMIN_PASSWORD")
if adminPass == "" {
    adminPass = "your_new_password"
}
if req.Password != adminPass {
```

### 修改数据库路径

编辑 `backend/main.go` 第 44 行附近：

```go
dbPath := os.Getenv("DB_PATH")
if dbPath == "" {
    dbPath = "./cards.db"
}
db, err = sql.Open("sqlite3", dbPath)
```

---

## 🧪 部署验证

### 1. 健康检查

```bash
# 检查服务是否运行
curl http://localhost:8080/api/health

# 预期响应
{"code":0,"message":"OK"}
```

### 2. 登录测试

```bash
# 测试登录接口
curl -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"password":"your_password"}'

# 预期响应
{"code":0,"message":"登录成功","data":{"token":"admin"}}
```

### 3. 前端访问

打开浏览器访问 `http://your-domain.com/query` 进行查询页面测试。

---

## 🔄 备份与恢复

### 自动备份

系统内置备份功能，通过管理后台可：
- 创建数据库备份
- 下载备份文件
- 恢复备份

### 手动备份

```bash
# 备份数据库
cp /var/lib/autocard/data/cards.db /backup/cards_$(date +%Y%m%d).db

# 恢复数据库
cp /backup/cards_20260317.db /var/lib/autocard/data/cards.db
```

---

## 🐛 常见问题

### Q1: 数据库文件丢失

**现象**: 容器重启后数据丢失  
**原因**: 未配置数据卷持久化  
**解决**: 确保正确挂载 `/data` 目录

### Q2: 前端无法连接 API

**现象**: 页面显示网络错误  
**原因**: 跨域或 API 地址配置错误  
**解决**: 
- 检查 `VITE_API_BASE_URL` 配置
- 确保后端 CORS 配置正确

### Q3: Railway 部署失败

**现象**: 构建或启动失败  
**解决**:
1. 检查 `railway.toml` 配置
2. 查看 Railway 日志排查错误
3. 确保 Dockerfile 可以本地构建成功

### Q4: 短信推送不生效

**现象**: 收不到短信验证码  
**排查**:
1. 检查 `/api/sms/push` 接口是否可访问
2. 查看后端日志确认推送接收
3. 验证短信格式是否匹配正则表达式

---

## 📊 监控与日志

### 查看日志

```bash
# Docker 日志
docker logs -f autocard

# Docker Compose
docker-compose logs -f

# Railway
# 在 Railway Dashboard 查看部署日志
```

### 性能监控

建议添加以下监控：
- 数据库大小增长
- API 响应时间
- 短信接收延迟

---

## 🔒 安全建议

1. **立即修改默认密码** - 不要使用 `jc123`
2. **启用 HTTPS** - 生产环境必须使用 HTTPS
3. **限制访问** - 使用防火墙限制端口访问
4. **定期备份** - 设置自动备份策略
5. **更新依赖** - 定期更新 Go 和 Node 依赖

---

## 📞 技术支持

如有部署问题，请检查：
1. 系统检查报告 `SYSTEM_CHECK_REPORT.md`
2. 技术文档 `技术文档.md`
3. 后端日志输出

---

*文档版本: v1.0*  
*最后更新: 2026-03-17*
