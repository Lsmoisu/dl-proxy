package middleware

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
)

// Recovery 中间件捕获任何panic并恢复
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 获取客户端IP
		clientIP := getClientIP(r)

		defer func() {
			if err := recover(); err != nil {
				// 记录错误和堆栈信息
				log.Printf("客户端: %s | 服务发生崩溃: %v\n堆栈追踪:\n%s",
					clientIP,
					err,
					debug.Stack())

				// 构建错误响应
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "内部服务器错误")
			}
		}()

		next.ServeHTTP(w, r)
	})
}
