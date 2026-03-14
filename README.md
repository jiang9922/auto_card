# Cards Manager

卡密管理系统 - 用于管理虚拟卡密（会员卡/充值卡等）

## 功能

- 批量导入卡密（格式：卡号----查询链接）
- 自动查询验证码
- 管理后台（筛选、分页、导出、删除）
- 用户查询页面

## 技术栈

- 后端：Go + Gin + SQLite
- 前端：Vue 3 + TypeScript + Vite

## 部署

### Railway 部署

1. Fork 本仓库到你的 GitHub
2. 登录 [Railway](https://railway.app)
3. 点击 "New Project" → "Deploy from GitHub repo"
4. 选择你的仓库
5. Railway 会自动识别 Dockerfile 并部署
6. 部署完成后会获得一个公网 URL

### 本地运行

```bash
# 后端
cd backend
go run main.go

# 前端
cd frontend
npm install
npm run dev
```

## 默认密码

- 管理后台密码：`admin123`