package middleware

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

// Logging 中间件记录每个请求的访问日志
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// 获取客户端IP
		clientIP := getClientIP(r)

		// 创建响应记录器
		recorder := &responseRecorder{
			ResponseWriter: w,
			status:         http.StatusOK,
			length:         0,
		}

		// 调用下一个处理器
		next.ServeHTTP(recorder, r)

		// 跳过对下载请求的重复日志记录，因为已经在handler中记录了详细信息
		if isDownloadRequest(r.URL.Path) {
			return
		}

		// 记录请求完成情况
		log.Printf(
			"客户端: %s | 路径: %s | 方法: %s | 状态: %d | 大小: %s | 耗时: %.2f秒",
			clientIP,
			r.URL.Path,
			r.Method,
			recorder.status,
			formatSize(recorder.length),
			time.Since(start).Seconds(),
		)
	})
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

// responseRecorder 包装http.ResponseWriter以记录响应状态和长度
type responseRecorder struct {
	http.ResponseWriter
	status int
	length int
}

// WriteHeader 重写以记录状态码
func (r *responseRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

// Write 重写以记录响应长度
func (r *responseRecorder) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.length += n
	return n, err
}

// formatSize 根据大小自动选择合适的单位
func formatSize(size int) string {
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
