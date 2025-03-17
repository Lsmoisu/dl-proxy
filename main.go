package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/yourusername/proxy-service/config"
	"github.com/yourusername/proxy-service/middleware"
	"github.com/yourusername/proxy-service/proxy"
	"github.com/yourusername/proxy-service/web"
)

var (
	configFile = flag.String("config", "config.yaml", "配置文件路径")
)

// logWriter 是一个自定义的日志写入器，用于添加时间戳
type logWriter struct {
	w io.Writer
}

// Write 实现 io.Writer 接口
func (lw *logWriter) Write(p []byte) (n int, err error) {
	// 添加时间戳
	timeStr := time.Now().Format("2006-01-02 15:04:05")
	logLine := fmt.Sprintf("[%s] %s", timeStr, p)
	return lw.w.Write([]byte(logLine))
}

func main() {
	flag.Parse()

	// 配置日志格式，包含时间戳
	log.SetFlags(log.Ldate | log.Ltime)

	// 自定义日志格式，使用更友好的时间格式
	log.SetFlags(0)   // 先清除所有标志
	log.SetPrefix("") // 清除前缀

	// 设置自定义的日志输出
	originalLogger := log.Default()
	log.SetOutput(&logWriter{originalLogger.Writer()})

	// 加载配置文件
	cfg, err := config.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 初始化请求限制器
	rateLimiter := proxy.NewRateLimiter(cfg.Security.RateLimiting.RequestsPerMinute)

	// 构建HTTP处理链
	handler := proxy.NewProxyHandler(cfg)

	// 注册静态资源和主页
	mux := http.NewServeMux()
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))
	mux.Handle("/", rootHandler(web.HomeHandler(), handler))
	mux.Handle("/health", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	// 所有其他请求都交给代理处理器
	mux.Handle("/*", handler)

	// 应用中间件
	wrappedHandler := middleware.Recovery(
		middleware.Logging(
			proxy.LimitRate(rateLimiter, mux),
		),
	)

	// 创建服务器
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      wrappedHandler,
		ReadTimeout:  time.Duration(cfg.Proxy.ConnectTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Proxy.TransferTimeout) * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 启动服务器
	go func() {
		log.Printf("代理服务器正在监听 %s:%d\n", cfg.Server.Host, cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v\n", err)
		}
	}()

	// 等待信号来优雅关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("正在关闭服务器...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器关闭失败: %v\n", err)
	}
	log.Println("服务器已优雅关闭")
}

// rootHandler 处理根路径请求，区分主页和代理请求
func rootHandler(homeHandler, proxyHandler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 只有确切的根路径才使用主页处理器
		if r.URL.Path == "/" {
			homeHandler.ServeHTTP(w, r)
			return
		}

		// 所有其他请求都交给代理处理器
		proxyHandler.ServeHTTP(w, r)
	})
}
