#!/bin/bash

# 定义变量
APP_NAME="dl-proxy"
PACKAGE_DIR="package"

# 创建输出目录
mkdir -p ${PACKAGE_DIR}
echo "开始多平台编译 ${APP_NAME}..."

# 清理旧文件
rm -rf ${PACKAGE_DIR}/*

# 编译 Windows AMD64 版本
echo "编译 Windows AMD64 版本..."
echo "输出文件: ${PACKAGE_DIR}/${APP_NAME}_windows_amd64.exe"
CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o ${PACKAGE_DIR}/${APP_NAME}_windows_amd64.exe || {
    echo "Windows AMD64 版本编译失败"
    exit 1
}

# 编译 Linux AMD64 版本
echo "编译 Linux AMD64 版本..."
echo "输出文件: ${PACKAGE_DIR}/${APP_NAME}_linux_amd64"
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${PACKAGE_DIR}/${APP_NAME}_linux_amd64 || {
    echo "Linux AMD64 版本编译失败"
    exit 1
}

# 编译 Linux ARM64 版本
echo "编译 Linux ARM64 版本..."
echo "输出文件: ${PACKAGE_DIR}/${APP_NAME}_linux_arm64"
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o ${PACKAGE_DIR}/${APP_NAME}_linux_arm64 || {
    echo "Linux ARM64 版本编译失败"
    exit 1
}

# 编译 macOS AMD64 版本
echo "编译 macOS AMD64 版本..."
echo "输出文件: ${PACKAGE_DIR}/${APP_NAME}_darwin_amd64"
CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o ${PACKAGE_DIR}/${APP_NAME}_darwin_amd64 || {
    echo "macOS AMD64 版本编译失败"
    exit 1
}

# 编译 macOS ARM64 版本 (Apple Silicon)
echo "编译 macOS ARM64 版本..."
echo "输出文件: ${PACKAGE_DIR}/${APP_NAME}_darwin_arm64"
CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -o ${PACKAGE_DIR}/${APP_NAME}_darwin_arm64 || {
    echo "macOS ARM64 版本编译失败"
    exit 1
}

# 添加可执行权限
chmod +x ${PACKAGE_DIR}/${APP_NAME}_linux_amd64
chmod +x ${PACKAGE_DIR}/${APP_NAME}_linux_arm64
chmod +x ${PACKAGE_DIR}/${APP_NAME}_darwin_amd64
chmod +x ${PACKAGE_DIR}/${APP_NAME}_darwin_arm64



echo "编译完成！所有文件已保存到 ${PACKAGE_DIR} 目录"
ls -lh ${PACKAGE_DIR} 
