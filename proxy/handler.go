package proxy

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/yourusername/proxy-service/config"
	"github.com/yourusername/proxy-service/utils"
)

const (
	maxResponseHeaderBytes = 1 << 20         // 1MB
	maxUrlLength           = 8 * 1024        // 8KB
	proxyIdentifier        = "GoStreamProxy" // 代理服务器标识
	nodeID                 = "node1"         // 节点标识
)

var (
	// 提取URL的正则表达式
	urlExtractor = regexp.MustCompile(`^/(https?:/?/?)([-a-zA-Z0-9@:%._\+~#=]{1,256}(?:\.[-a-zA-Z0-9()]{1,6})+(?:[-a-zA-Z0-9()@:%_\+.~#?&//=]*))$`)

	// RFC1918 私有地址检测正则
	privateIPRegex = regexp.MustCompile(`^(127\.|10\.|172\.(1[6-9]|2[0-9]|3[0-1])\.|192\.168\.)`)

	// 内存缓冲池
	bufferPool = utils.NewBufferPool(32 * 1024) // 32KB 缓冲区

	// 下载跟踪器，用于合并分片下载的日志
	downloadTracker = NewDownloadTracker()

	// 需要删除的敏感头列表
	sensitiveHeaders = []string{
		"Authorization",
		"Cookie",
		"Set-Cookie",
	}
)

// DownloadTracker 用于跟踪文件下载状态
type DownloadTracker struct {
	downloads map[string]*DownloadInfo
	mu        sync.RWMutex
}

// DownloadInfo 存储下载信息
type DownloadInfo struct {
	fileName     string
	totalSize    int64
	bytesWritten int64
	startTime    time.Time
	lastLogTime  time.Time
	isComplete   bool
	activeConns  int
	clientIP     string
	mu           sync.Mutex
}

// NewDownloadTracker 创建新的下载跟踪器
func NewDownloadTracker() *DownloadTracker {
	return &DownloadTracker{
		downloads: make(map[string]*DownloadInfo),
	}
}

// GetOrCreate 获取或创建下载信息
func (dt *DownloadTracker) GetOrCreate(url string, fileName string, totalSize int64, clientIP string) *DownloadInfo {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	// 使用URL作为唯一标识
	key := url

	info, exists := dt.downloads[key]
	if !exists {
		info = &DownloadInfo{
			fileName:     fileName,
			totalSize:    totalSize,
			bytesWritten: 0,
			startTime:    time.Now(),
			lastLogTime:  time.Now(),
			isComplete:   false,
			activeConns:  0,
			clientIP:     clientIP,
		}
		dt.downloads[key] = info

		// 记录下载开始
		log.Printf("客户端: %s | 开始下载: %s, 预计大小: %s",
			clientIP,
			fileName,
			formatFileSize(totalSize))
	}

	// 增加活跃连接计数
	info.mu.Lock()
	info.activeConns++
	info.mu.Unlock()

	return info
}

// UpdateProgress 更新下载进度
func (di *DownloadInfo) UpdateProgress(bytesAdded int64) {
	di.mu.Lock()
	defer di.mu.Unlock()

	di.bytesWritten += bytesAdded

	// 每10秒记录一次进度
	now := time.Now()
	if now.Sub(di.lastLogTime) >= 10*time.Second && !di.isComplete {
		di.logProgress()
		di.lastLogTime = now
	}
}

// logProgress 记录当前下载进度
func (di *DownloadInfo) logProgress() {
	elapsedTime := time.Since(di.startTime)
	speedMBps := float64(di.bytesWritten) / elapsedTime.Seconds() / 1024 / 1024

	if di.totalSize > 0 {
		percentage := float64(di.bytesWritten) * 100 / float64(di.totalSize)
		log.Printf("客户端: %s | 下载进度: %s, %.2f%% (已下载: %s / 总大小: %s), 速度: %.2f MB/s, 活跃连接: %d",
			di.clientIP,
			di.fileName,
			percentage,
			formatFileSize(di.bytesWritten),
			formatFileSize(di.totalSize),
			speedMBps,
			di.activeConns)
	} else {
		log.Printf("客户端: %s | 下载进度: %s, 已下载: %s, 速度: %.2f MB/s, 活跃连接: %d",
			di.clientIP,
			di.fileName,
			formatFileSize(di.bytesWritten),
			speedMBps,
			di.activeConns)
	}
}

// ConnectionClosed 标记一个连接已关闭
func (dt *DownloadTracker) ConnectionClosed(url string, err error) bool {
	dt.mu.RLock()
	info, exists := dt.downloads[url]
	dt.mu.RUnlock()

	if !exists {
		return false
	}

	info.mu.Lock()
	defer info.mu.Unlock()

	// 减少活跃连接计数
	info.activeConns--

	// 如果还有活跃连接，则不是最终状态
	if info.activeConns > 0 {
		return false
	}

	// 所有连接都已关闭，判断是否完成下载
	if err == nil || info.bytesWritten >= info.totalSize {
		// 下载完成
		if !info.isComplete {
			info.isComplete = true
			downloadDuration := time.Since(info.startTime)
			speedMBps := float64(info.bytesWritten) / downloadDuration.Seconds() / 1024 / 1024

			log.Printf("客户端: %s | 下载完成: %s, 大小: %s, 耗时: %.2f秒, 速度: %.2f MB/s",
				info.clientIP,
				info.fileName,
				formatFileSize(info.bytesWritten),
				downloadDuration.Seconds(),
				speedMBps)
		}
		return true
	} else if isClientDisconnectError(err) {
		// 客户端取消下载
		if !info.isComplete {
			info.isComplete = true
			log.Printf("客户端: %s | 下载取消: %s - 客户端断开连接",
				info.clientIP,
				info.fileName)
		}
		return true
	} else {
		// 下载出错
		if !info.isComplete {
			info.isComplete = true
			log.Printf("客户端: %s | 下载错误: %s - %v",
				info.clientIP,
				info.fileName,
				err)
		}
		return true
	}
}

// CleanupOldDownloads 清理完成超过一定时间的下载记录
func (dt *DownloadTracker) CleanupOldDownloads() {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	threshold := time.Now().Add(-30 * time.Minute)
	for url, info := range dt.downloads {
		info.mu.Lock()
		if info.isComplete && info.lastLogTime.Before(threshold) {
			delete(dt.downloads, url)
		}
		info.mu.Unlock()
	}
}

type ProxyHandler struct {
	client *http.Client
	config *config.Config
}

// NewProxyHandler 创建新的代理处理器
func NewProxyHandler(cfg *config.Config) *ProxyHandler {
	// 配置传输层
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   time.Duration(cfg.Proxy.ConnectTimeout) * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   time.Duration(cfg.Proxy.ConnectTimeout) * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: time.Duration(cfg.Proxy.ConnectTimeout) * time.Second,
		// 允许不安全TLS连接以方便代理
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	// 创建客户端
	client := &http.Client{
		Transport: transport,
		Timeout:   time.Duration(cfg.Proxy.TransferTimeout) * time.Second,
	}

	return &ProxyHandler{
		client: client,
		config: cfg,
	}
}

// ServeHTTP 实现http.Handler接口
func (p *ProxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 获取客户端IP地址
	clientIP := getClientIP(r)

	// 判断是否为下载请求
	isDownload := isDownloadRequest(r.URL.Path)

	// 只记录非下载请求的基本信息，下载请求会在后续处理中记录
	if !isDownload {
		log.Printf("客户端: %s | 请求: %s %s",
			clientIP,
			r.Method,
			r.URL.Path)
	}

	// 健康检查端点保持单独处理
	if r.URL.Path == "/health" {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
		return
	}

	// 如果不是下载请求，直接返回404或重定向到主页
	if !isDownload && r.URL.Path != "/" {
		// 对于favicon.ico等常见请求，返回404而不是错误信息
		if r.URL.Path == "/favicon.ico" {
			http.NotFound(w, r)
			return
		}

		// 其他非下载请求返回错误
		http.Error(w, "无效的请求路径", http.StatusBadRequest)
		return
	}

	// 如果是主页请求，交给其他处理器处理
	if r.URL.Path == "/" {
		return
	}

	// 提取目标URL
	targetURL, err := p.extractTargetURL(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("无效的URL: %v", err), http.StatusBadRequest)
		log.Printf("客户端: %s | 错误: 无效的URL: %v",
			clientIP,
			err)
		return
	}

	// 验证URL格式
	if err := ValidateURL(targetURL); err != nil {
		http.Error(w, fmt.Sprintf("URL验证失败: %v", err), http.StatusBadRequest)
		log.Printf("客户端: %s | 错误: URL验证失败: %v",
			clientIP,
			err)
		return
	}

	// 检查是否为内网地址
	if isPrivateIP(targetURL) {
		http.Error(w, "不允许访问内网地址", http.StatusForbidden)
		log.Printf("客户端: %s | 错误: 尝试访问内网地址: %s",
			clientIP,
			targetURL.String())
		return
	}

	// 设置请求超时
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(p.config.Proxy.TransferTimeout)*time.Second)
	defer cancel()
	r = r.WithContext(ctx)

	// 创建代理请求
	proxyReq, err, reqCancel := p.createProxyRequest(r, targetURL)
	if reqCancel != nil {
		defer reqCancel()
	}
	if err != nil {
		http.Error(w, fmt.Sprintf("创建代理请求失败: %v", err), http.StatusInternalServerError)
		log.Printf("客户端: %s | 错误: 创建代理请求失败: %v",
			clientIP,
			err)
		return
	}

	// 处理请求头
	processRequestHeadersInternal(proxyReq)

	// 从URL中提取文件名
	fileName := extractFilenameFromURL(targetURL)

	// 执行代理请求
	resp, err := p.client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("代理请求失败: %v", err), http.StatusBadGateway)
		log.Printf("客户端: %s | 错误: 代理请求失败: %v",
			clientIP,
			err)
		downloadTracker.ConnectionClosed(targetURL.String(), err)
		return
	}
	defer resp.Body.Close()

	// 处理响应头
	processResponseHeadersInternal(w, resp, time.Since(time.Now()))
	// 确保文件下载头
	ensureDownloadHeaders(w, resp, targetURL)

	// 获取并处理Content-Disposition头
	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		// 保留原始的Content-Disposition头
		w.Header().Set("Content-Disposition", contentDisposition)
	} else {
		// 如果目标服务器没有提供Content-Disposition，尝试从URL中提取文件名
		if fileName != "" {
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fileName))
		}
	}

	// 获取文件大小
	fileSize := resp.ContentLength

	// 转发响应状态码
	w.WriteHeader(resp.StatusCode)

	// 使用内存池缓冲区传输响应体
	buffer := bufferPool.Get()
	defer bufferPool.Put(buffer)

	// 获取或创建下载信息
	downloadInfo := downloadTracker.GetOrCreate(targetURL.String(), fileName, fileSize, clientIP)

	// 创建自定义写入器
	writer := &trackedWriter{
		w:            w,
		downloadInfo: downloadInfo,
	}

	// 流式传输响应体
	_, err = io.CopyBuffer(writer, resp.Body, buffer)

	// 处理下载完成或错误
	downloadTracker.ConnectionClosed(targetURL.String(), err)
}

// trackedWriter 是一个包装了http.ResponseWriter的结构，用于跟踪下载进度
type trackedWriter struct {
	w            http.ResponseWriter
	downloadInfo *DownloadInfo
}

// Write 实现io.Writer接口
func (tw *trackedWriter) Write(p []byte) (int, error) {
	n, err := tw.w.Write(p)
	if err != nil {
		return n, err
	}

	// 更新下载进度
	tw.downloadInfo.UpdateProgress(int64(n))

	return n, nil
}

// 定期清理旧的下载记录
func init() {
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for range ticker.C {
			downloadTracker.CleanupOldDownloads()
		}
	}()
}

// isClientDisconnectError 检查错误是否为客户端断开连接
func isClientDisconnectError(err error) bool {
	if err == nil {
		return false
	}

	errStr := err.Error()
	return strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "reset by peer") ||
		strings.Contains(errStr, "client disconnected") ||
		strings.Contains(errStr, "connection reset") ||
		strings.Contains(errStr, "context canceled")
}

// extractTargetURL 从请求中提取目标URL
func (p *ProxyHandler) extractTargetURL(r *http.Request) (*url.URL, error) {
	path := r.URL.Path

	// 如果URL太长，直接拒绝
	if len(path) > maxUrlLength {
		return nil, fmt.Errorf("URL过长(最大支持%d字节)", maxUrlLength)
	}

	matches := urlExtractor.FindStringSubmatch(path)
	if len(matches) < 3 {
		return nil, fmt.Errorf("无法从路径提取目标URL: %s", path)
	}

	// 确保协议后有两个斜杠
	protocol := matches[1]
	if protocol == "https:/" || protocol == "http:/" {
		protocol = protocol + "/"
	}

	// 重建完整URL
	targetURLStr := protocol + matches[2]

	// 保留原始查询参数
	if r.URL.RawQuery != "" {
		targetURLStr += "?" + r.URL.RawQuery
	}

	return url.Parse(targetURLStr)
}

// createProxyRequest 创建代理请求
func (p *ProxyHandler) createProxyRequest(r *http.Request, targetURL *url.URL) (*http.Request, error, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Duration(p.config.Proxy.TransferTimeout)*time.Second)

	proxyReq, err := http.NewRequestWithContext(
		ctx,
		r.Method,
		targetURL.String(),
		nil,
	)
	if err != nil {
		cancel() // 如果出错立即取消
		return nil, err, nil
	}

	// 复制原始请求头
	for k, vv := range r.Header {
		// 跳过Connection相关头，让HTTP客户端自行管理
		if strings.EqualFold(k, "Connection") ||
			strings.EqualFold(k, "Keep-Alive") ||
			strings.EqualFold(k, "Proxy-Connection") ||
			strings.EqualFold(k, "Transfer-Encoding") ||
			strings.EqualFold(k, "Upgrade") {
			continue
		}
		for _, v := range vv {
			proxyReq.Header.Add(k, v)
		}
	}

	// 设置X-Forwarded-For
	if clientIP, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		if prior, ok := proxyReq.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		proxyReq.Header.Set("X-Forwarded-For", clientIP)
	}

	return proxyReq, nil, cancel
}

// isPrivateIP 检查URL是否指向私有IP地址
func isPrivateIP(targetURL *url.URL) bool {
	host := targetURL.Hostname()

	// 如果是IP地址，直接检查
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsPrivate()
	}

	// 解析域名为IP
	addrs, err := net.LookupHost(host)
	if err != nil {
		return false
	}

	// 检查解析出的所有IP
	for _, addr := range addrs {
		ip := net.ParseIP(addr)
		if ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
			return true
		}

		// 使用正则表达式检查是否为RFC1918地址
		if privateIPRegex.MatchString(addr) {
			return true
		}
	}

	return false
}

// 确保正确设置文件下载头
func ensureDownloadHeaders(w http.ResponseWriter, resp *http.Response, targetURL *url.URL) {
	// 从URL路径中提取文件名
	urlPath := targetURL.Path
	fileName := urlPath[strings.LastIndex(urlPath, "/")+1:]

	// 复制内容类型，若原始响应有设置
	contentType := resp.Header.Get("Content-Type")
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	} else {
		// 根据文件扩展名推断内容类型
		ext := strings.ToLower(filepath.Ext(fileName))
		switch ext {
		case ".zip":
			w.Header().Set("Content-Type", "application/zip")
		case ".exe":
			w.Header().Set("Content-Type", "application/octet-stream")
		case ".pdf":
			w.Header().Set("Content-Type", "application/pdf")
		default:
			w.Header().Set("Content-Type", "application/octet-stream")
		}
	}

	// 设置文件大小，若原始响应有设置
	contentLength := resp.Header.Get("Content-Length")
	if contentLength != "" {
		w.Header().Set("Content-Length", contentLength)
	}

	// 添加额外头，防止浏览器缓存
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
}

// 添加此辅助函数用于从URL中提取文件名
func extractFilenameFromURL(targetURL *url.URL) string {
	// 首先尝试从查询参数中提取文件名
	queryParams := targetURL.Query()
	if filename := queryParams.Get("filename"); filename != "" {
		return filename
	}

	// 查找response-content-disposition参数
	if disposition := queryParams.Get("response-content-disposition"); disposition != "" {
		// 解析disposition值
		if strings.Contains(disposition, "filename=") {
			parts := strings.Split(disposition, "filename=")
			if len(parts) > 1 {
				filename := parts[1]
				// 移除可能的引号
				filename = strings.Trim(filename, "\"'")
				return filename
			}
		}
	}

	// 从URL路径中提取文件名作为后备方案
	path := targetURL.Path
	segments := strings.Split(path, "/")
	if len(segments) > 0 {
		lastSegment := segments[len(segments)-1]
		if lastSegment != "" {
			return lastSegment
		}
	}

	return ""
}

// formatFileSize 根据文件大小自动选择合适的单位
func formatFileSize(size int64) string {
	if size < 0 {
		return "未知大小"
	}

	const (
		B  = 1
		KB = 1024 * B
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case size >= TB:
		return fmt.Sprintf("%.2f TB", float64(size)/TB)
	case size >= GB:
		return fmt.Sprintf("%.2f GB", float64(size)/GB)
	case size >= MB:
		return fmt.Sprintf("%.2f MB", float64(size)/MB)
	case size >= KB:
		return fmt.Sprintf("%.2f KB", float64(size)/KB)
	default:
		return fmt.Sprintf("%d B", size)
	}
}

// getClientIP 获取客户端真实IP地址
func getClientIP(r *http.Request) string {
	// 首先尝试从X-Forwarded-For头获取
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		// 取第一个IP（最原始的客户端IP）
		if len(ips) > 0 {
			clientIP := strings.TrimSpace(ips[0])
			if clientIP != "" {
				return clientIP
			}
		}
	}

	// 尝试从X-Real-IP头获取
	if xrip := r.Header.Get("X-Real-IP"); xrip != "" {
		return strings.TrimSpace(xrip)
	}

	// 从RemoteAddr获取
	if r.RemoteAddr != "" {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil {
			return ip
		}
		return r.RemoteAddr
	}

	return "未知IP"
}

// isDownloadRequest 判断是否为下载请求
func isDownloadRequest(path string) bool {
	return strings.HasPrefix(path, "/http:/") ||
		strings.HasPrefix(path, "/https:/")
}

// 重命名这些函数以避免冲突
func processRequestHeadersInternal(r *http.Request) {
	// 删除Proxy-*系列头
	for header := range r.Header {
		if strings.HasPrefix(strings.ToLower(header), "proxy-") {
			r.Header.Del(header)
		}
	}

	// 重写Via头
	if via := r.Header.Get("Via"); via != "" {
		r.Header.Set("Via", via+", "+proxyIdentifier)
	} else {
		r.Header.Set("Via", r.Proto+" "+proxyIdentifier)
	}

	// 清理敏感头
	for _, header := range sensitiveHeaders {
		r.Header.Del(header)
	}
}

// 重命名这些函数以避免冲突
func processResponseHeadersInternal(w http.ResponseWriter, resp *http.Response, duration time.Duration) {
	// 复制所有响应头
	for header, values := range resp.Header {
		// 跳过敏感头
		if containsInternal(sensitiveHeaders, header) {
			continue
		}

		for _, value := range values {
			w.Header().Add(header, value)
		}
	}

	// 添加安全头
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("X-Frame-Options", "DENY")

	// 如果是HTTPS请求，添加HSTS头
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

	// 添加CSP头
	w.Header().Set("Content-Security-Policy", "default-src 'self'")

	// 添加处理时间
	w.Header().Set("X-Processing-Time", fmt.Sprintf("%.6fs", duration.Seconds()))
	w.Header().Set("X-Proxy-Node", nodeID)
}

// 重命名这个函数以避免冲突
func containsInternal(slice []string, item string) bool {
	item = strings.ToLower(item)
	for _, s := range slice {
		if strings.ToLower(s) == item {
			return true
		}
	}
	return false
}
