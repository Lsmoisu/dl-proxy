# 文件下载代理加速 [![GitHub](https://img.shields.io/badge/GitHub-Open_Source-181717?logo=github)](https://github.com/Lsmoisu/dl-proxy)

[![GitHub stars](https://img.shields.io/github/stars/Lsmoisu/dl-proxy?style=flat-square)](https://github.com/Lsmoisu/dl-proxy/stargazers)
[![GitHub release](https://img.shields.io/github/v/release/Lsmoisu/dl-proxy?include_prereleases&style=flat-square)](https://github.com/Lsmoisu/dl-proxy/releases)
[![Docker Pulls](https://img.shields.io/docker/pulls/zarilla/dl-proxy?style=flat-square)](https://hub.docker.com/r/zarilla/dl-proxy)
[![License](https://img.shields.io/badge/License-MIT-blue.svg?style=flat-square)](LICENSE)

<a href="https://github.com/Lsmoisu/dl-proxy" target="_blank">
  <img src="https://github.githubassets.com/images/modules/logos_page/GitHub-Mark.png" 
       alt="GitHub Repository" 
       width="32" 
       style="vertical-align:middle;margin-right:10px;">
</a>

**开源声明**：本项目采用 MIT 协议开源，欢迎自由使用和二次开发。如果对您有帮助，欢迎 ⭐Star 支持！

这是一个高性能的 HTTP 流式传输代理服务，支持大文件下载和安全过滤。

## 功能特性

- **流式传输**：支持大文件的零缓存高效下载
- **安全过滤**：自动屏蔽危险请求
- **高性能**：支持高并发和大文件传输

## 示例网站

示例网站：
https://dl.aaa.team/
https://df.aaa.team/

## 安装与部署

### 传统编译方式

#### 环境要求
- Go 1.21+
- Git

```bash
git clone https://github.com/Lsmoisu/dl-proxy.git
cd dl-proxy
go build -o dl-proxy
```

### Docker部署

#### 快速启动
```bash
docker run -d \
  -p 18080:8080 \
  -v ./config.yaml:/app/config.yaml:ro \
  --name dl-proxy \
  zarilla/dl-proxy:latest
```

#### 多架构构建
```bash
# 构建并推送镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t your-username/dl-proxy:latest \
  --push .
```

### 使用Docker Compose
创建docker-compose.yml文件后执行：
```bash
docker-compose up -d
```

### 自动部署（GitHub Actions）
1. 创建GitHub Release或推送tag
2. 镜像将自动构建并推送到GitHub Container Registry
3. 拉取最新镜像：
```bash
docker pull ghcr.io/你的用户名/dl-proxy:latest
```

## 配置说明

你可以通过 `config.yaml` 文件自定义服务配置，包括端口、超时设置等。

### 示例配置

```yaml
server:
  host: "0.0.0.0"  # 监听地址，"0.0.0.0" 表示监听所有网络接口
  port: 8080       # 监听端口，服务将运行在这个端口上

proxy:
  connectTimeout: 5          # 连接超时（秒），设置与目标服务器的连接超时时间
  transferTimeout: 300       # 传输超时（秒），设置文件传输的最大允许时间
  bufferSize: 32768          # 缓冲区大小（字节），用于流式传输的内存缓冲区大小
  chunkedThreshold: 104857600 # 分块传输阈值（字节），超过此大小的文件将使用分块传输

security:
  rateLimiting:
    enabled: true            # 是否启用请求频率限制
    requestsPerMinute: 60    # 每IP每分钟允许的最大请求数
  privateIPBlocking: true    # 是否阻止对内网IP地址的请求
```

### 配置项详细说明

- **server**: 配置服务的监听地址和端口。
- **proxy**: 配置代理的连接和传输超时、缓冲区大小等。
- **security**: 配置安全相关的选项，如请求频率限制和内网IP阻止。

## 性能指标
- 吞吐量：≥800MB/s
- 延迟波动：<±5%
- 支持1000+连接连接

## 更新日志
### v1.1.0
- 新增Docker多架构支持
- 优化内存管理
- 增加健康检查接口

## 法律风险提示

使用本服务时，请确保遵守相关法律法规。用户需自行承担因使用本服务而产生的任何法律责任。请勿使用本服务进行任何非法活动，包括但不限于下载受版权保护的内容。

## 部署方法

1. 确保服务器上已安装 Go 和 Git。
2. 克隆项目并进入项目目录。
3. 编译项目：`go build -o dl-proxy`
4. 运行服务：`./dl-proxy`
5. 确保防火墙允许服务监听的端口。

## 依赖

- `gopkg.in/yaml.v3`: 用于解析 YAML 配置文件。
- `github.com/google/uuid`: 用于生成唯一请求ID。

安装依赖：

```bash
go get gopkg.in/yaml.v3
go get github.com/google/uuid
```

## 常见问题

### 无法下载文件

- 确保输入的 URL 是完整且有效的
- 检查服务日志，查看是否有错误信息

### 服务无法启动

- 确保端口未被占用
- 检查配置文件格式是否正确

## 贡献

欢迎提交问题和请求合并。请确保在提交前运行所有测试。

## 许可证

MIT License
