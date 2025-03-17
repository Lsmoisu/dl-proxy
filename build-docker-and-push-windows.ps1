# 设置镜像名称和标签
$imageName = "zarilla/dl-proxy-p"
$tag = "latest"

# 设置目标平台
$platforms = "linux/amd64,linux/arm64"

# 确保buildx可用
docker buildx inspect --bootstrap

# 创建新的构建器实例（如果需要）
if (-not (docker buildx ls | Select-String -Pattern "mybuilder")) {
  docker buildx create --name mybuilder --use
}

# 使用buildx构建多架构镜像
Write-Host "Building multi-architecture Docker image..."
docker buildx build --platform $platforms `
  -t "${imageName}:${tag}" `
  --push .

Write-Host "Multi-architecture Docker image built and pushed successfully!" 