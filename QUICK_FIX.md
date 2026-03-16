# Autocard 项目上线前快速修复清单

## ⚡ 紧急修复（必须完成）

### 1. 修复硬编码密码

**文件**: `backend/main.go` 第 116 行

```bash
# 使用 sed 替换（在 Linux/Mac 上）
sed -i 's/if req.Password != "jc123"/adminPassword := os.Getenv("ADMIN_PASSWORD")\n\tif adminPassword == "" {\n\t\tadminPassword = "jc123"\n\t}\n\tif req.Password != adminPassword/' backend/main.go
```

**手动修改**: 查看 `backend/main.go.patch` 获取完整代码

### 2. 修复数据库路径

**文件**: `backend/main.go` 第 44 行

```go
// 修改前
db, err = sql.Open("sqlite3", "./cards.db")

// 修改后
dbPath := os.Getenv("DB_PATH")
if dbPath == "" {
    dbPath = "./cards.db"
}
db, err = sql.Open("sqlite3", dbPath)
```

### 3. 更新 Dockerfile

**文件**: `Dockerfile`

```bash
# 备份原文件
cp Dockerfile Dockerfile.bak

# 使用修复版本
cp Dockerfile.fixed Dockerfile
```

**关键修改**:
- 将 `alpine:latest` 改为 `alpine:3.19`
- 添加 `VOLUME ["/data"]`
- 添加 `ENV DB_PATH=/data/cards.db`
- 添加 HEALTHCHECK

### 4. 更新 Railway 配置

**文件**: `railway.toml`

```bash
cp railway.toml railway.toml.bak
cp railway.toml.fixed railway.toml
```

### 5. 添加前端生产环境配置

**文件**: `frontend/.env.production`

已创建，无需额外操作。

---

## 🔧 部署前检查清单

### 本地测试

```bash
# 1. 构建 Docker 镜像
docker build -t autocard:test .

# 2. 运行测试容器
docker run -d \
  --name autocard-test \
  -p 8080:8080 \
  -e PORT=8080 \
  -e ADMIN_PASSWORD=test123 \
  -e DB_PATH=/data/cards.db \
  -v $(pwd)/test-data:/data \
  autocard:test

# 3. 等待启动
sleep 5

# 4. 健康检查
curl http://localhost:8080/api/health

# 5. 登录测试
curl -X POST http://localhost:8080/api/admin/login \
  -H "Content-Type: application/json" \
  -d '{"password":"test123"}'

# 6. 清理测试容器
docker stop autocard-test
docker rm autocard-test
```

### Railway 部署检查

```bash
# 确保环境变量已设置
railway variables

# 必需变量:
# - PORT=8080
# - ADMIN_PASSWORD=your_secure_password
# - DB_PATH=/data/cards.db
```

---

## 📋 文件清单

修复后应包含以下文件：

```
auto_card/
├── Dockerfile                    # ✅ 已修复
├── Dockerfile.bak               # 原文件备份
├── Dockerfile.fixed             # 修复模板
├── railway.toml                 # ✅ 已修复
├── railway.toml.bak            # 原文件备份
├── railway.toml.fixed          # 修复模板
├── SYSTEM_CHECK_REPORT.md      # 系统检查报告
├── DEPLOY.md                   # 部署文档
├── QUICK_FIX.md               # 本文件
├── backend/
│   ├── main.go                # ✅ 需要修改
│   └── main.go.patch          # 代码修复参考
└── frontend/
    └── .env.production        # ✅ 已创建
```

---

## 🚀 一键修复脚本

```bash
#!/bin/bash
# save_fix.sh - 保存修复建议的脚本

echo "=== Autocard 项目修复脚本 ==="

# 备份原文件
cp Dockerfile Dockerfile.bak 2>/dev/null || true
cp railway.toml railway.toml.bak 2>/dev/null || true

# 应用修复
cp Dockerfile.fixed Dockerfile
cp railway.toml.fixed railway.toml

echo "✅ Docker 和 Railway 配置已修复"
echo ""
echo "⚠️  还需要手动修复 backend/main.go:"
echo "   1. 搜索 'jc123' 并添加环境变量支持"
echo "   2. 搜索 './cards.db' 并添加 DB_PATH 环境变量支持"
echo ""
echo "📖 参考文件:"
echo "   - backend/main.go.patch"
echo "   - DEPLOY.md"
echo ""
echo "完成后请运行本地测试验证修复"
```

---

## ❓ 常见问题快速解决

### Q: 修改后编译失败？

```bash
# 进入 backend 目录
cd backend

# 确保依赖完整
go mod tidy

# 本地编译测试
go build -o test-server main.go
```

### Q: 数据库权限错误？

```bash
# 确保数据目录有正确权限
chmod 755 /data
chown -R 1000:1000 /data
```

### Q: 前端 API 调用失败？

检查 `frontend/.env.production` 中的 `VITE_API_BASE_URL` 配置：
- 同域部署：留空
- 跨域部署：填写完整 API 地址

---

## ✅ 上线前最终确认

- [ ] 管理员密码已修改为强密码
- [ ] Dockerfile 已添加数据卷和 HEALTHCHECK
- [ ] Railway/VPS 已配置数据持久化
- [ ] 本地测试通过
- [ ] 备份策略已制定
- [ ] HTTPS 已配置（生产环境）

---

*快速修复清单 v1.0*  
*生成时间: 2026-03-17*
