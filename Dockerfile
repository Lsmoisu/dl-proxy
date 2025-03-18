# 使用官方 golang 镜像作为构建阶段，支持多架构
FROM --platform=$BUILDPLATFORM golang:1.21 AS builder

# 设置工作目录
WORKDIR /app

# 复制 Go 模块文件并下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制项目所有文件
COPY . .

# 设置目标架构环境变量，由 docker buildx 注入
ARG TARGETOS
ARG TARGETARCH

# 编译 Go 可执行文件，启用 CGO=0 并指定目标架构
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o dl-proxy main.go

# 使用轻量级的 alpine 镜像作为最终运行阶段
FROM alpine:latest

# 设置工作目录
WORKDIR /app

# 从构建阶段复制编译好的可执行文件
COPY --from=builder /app/dl-proxy .

# 复制前端资源和配置文件
COPY web/ ./web/
COPY config.yaml .

# 安装必要的运行时依赖并设置东八区时区
RUN apk --no-cache add tzdata ca-certificates && \
    cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && \
    echo "Asia/Shanghai" > /etc/timezone

# 设置容器启动时运行的可执行文件
ENTRYPOINT ["/app/dl-proxy"]

# 默认暴露端口（根据需要调整）
EXPOSE 8080
