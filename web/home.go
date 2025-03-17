package web

import (
	"html/template"
	"net/http"
	"path/filepath"
)

// HomeHandler 处理主页请求
func HomeHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// 非根路径请求，交给代理处理器
			return
		}

		// 解析模板
		tmpl, err := template.ParseFiles(filepath.Join("web", "templates", "home.html"))
		if err != nil {
			http.Error(w, "模板解析失败", http.StatusInternalServerError)
			return
		}

		// 渲染模板
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := tmpl.Execute(w, nil); err != nil {
			http.Error(w, "模板渲染失败", http.StatusInternalServerError)
			return
		}
	})
}
