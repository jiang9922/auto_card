# 构建前端
FROM node:20-alpine AS frontend-builder
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm install
COPY frontend/ ./
RUN npm run build-only

# 构建后端
FROM golang:1.22-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev sqlite-dev
WORKDIR /app/backend
# 缓存破坏 - 强制重新构建
ARG CACHE_BUST=45
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o cards-server main.go

# 最终镜像
FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

# 复制后端可执行文件
COPY --from=backend-builder /app/backend/cards-server .

# 复制前端构建产物
COPY --from=frontend-builder /app/frontend/dist ./frontend/dist

# 创建数据库目录
RUN mkdir -p /data

# 设置环境变量
ENV PORT=8080
ENV GIN_MODE=release

# 暴露端口
EXPOSE 8080

# 启动命令
CMD ["./cards-server"]