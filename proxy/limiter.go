package proxy

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// 自定义HTTP错误码
const (
	StatusTooManyRequests        = 429 // RFC 6585, 4.4
	StatusBandwidthLimitExceeded = 509 // 非标准，但常用
)

// RateLimiter 使用滑动窗口限制请求频率
type RateLimiter struct {
	windows     map[string]*slidingWindow
	windowSize  time.Duration
	maxRequests int
	mu          sync.RWMutex
}

// slidingWindow 记录时间窗口内的请求
type slidingWindow struct {
	timestamps []time.Time
	index      int
	size       int
	lastClean  time.Time
}

// NewRateLimiter 创建一个新的频率限制器
// 默认每IP每分钟允许60个请求
func NewRateLimiter(requestsPerMinute int) *RateLimiter {
	return &RateLimiter{
		windows:     make(map[string]*slidingWindow),
		windowSize:  time.Minute,
		maxRequests: requestsPerMinute,
	}
}

// Allow 判断是否允许请求
func (rl *RateLimiter) Allow(ip string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// 获取该IP的窗口，如果不存在则创建
	window, exists := rl.windows[ip]
	if !exists {
		window = &slidingWindow{
			timestamps: make([]time.Time, rl.maxRequests),
			size:       rl.maxRequests,
			lastClean:  now,
		}
		rl.windows[ip] = window
	}

	// 清理过期记录
	if now.Sub(window.lastClean) > time.Minute {
		rl.cleanupOldRecords(window, now)
	}

	// 获取窗口最老的时间戳
	oldestTime := window.timestamps[(window.index+1)%window.size]

	// 如果最老的记录在窗口期内且总数已达上限，则拒绝请求
	if !oldestTime.IsZero() && now.Sub(oldestTime) <= rl.windowSize && countNonZero(window.timestamps) >= window.size {
		return false
	}

	// 记录当前请求时间戳
	window.timestamps[window.index] = now
	window.index = (window.index + 1) % window.size

	return true
}

// 清理过期记录
func (rl *RateLimiter) cleanupOldRecords(window *slidingWindow, now time.Time) {
	// 使用环形缓冲区，只清理头部过期的记录
	for !window.timestamps[window.index].IsZero() && window.timestamps[window.index].Before(now.Add(-rl.windowSize)) {
		window.timestamps[window.index] = time.Time{}
		window.index = (window.index + 1) % window.size
	}
	window.lastClean = now
}

// 计算非零时间戳数量
func countNonZero(timestamps []time.Time) int {
	count := 0
	for _, t := range timestamps {
		if !t.IsZero() {
			count++
		}
	}
	return count
}

// LimitRate 是一个中间件，用于在请求级别限制频率
func LimitRate(limiter *RateLimiter, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 提取客户端IP
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			ip = r.RemoteAddr
		}

		// 检查是否允许请求
		if !limiter.Allow(ip) {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "请求频率超限", StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}
