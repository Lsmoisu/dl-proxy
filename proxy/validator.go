package proxy

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	// 有效URL格式验证正则表达式
	validURLRegex = regexp.MustCompile(`^https?://[-a-zA-Z0-9@:%._\+~#=]{1,256}\.[a-zA-Z0-9()]{1,6}\b(?:[-a-zA-Z0-9()@:%_\+.~#?&//=]*)$`)
)

// ValidateURL 验证URL格式是否有效
func ValidateURL(targetURL *url.URL) error {
	// 对于常见的可信网站直接允许
	if strings.HasSuffix(targetURL.Host, "github.com") ||
		strings.HasSuffix(targetURL.Host, "githubusercontent.com") {
		return nil
	}

	urlStr := targetURL.String()

	// 检查协议
	if targetURL.Scheme != "http" && targetURL.Scheme != "https" {
		return fmt.Errorf("不支持的协议: %s", targetURL.Scheme)
	}

	// 检查主机名
	if targetURL.Host == "" {
		return fmt.Errorf("主机名不能为空")
	}

	// 使用正则表达式检查完整URL格式
	if !validURLRegex.MatchString(urlStr) {
		return fmt.Errorf("URL格式无效: %s", urlStr)
	}

	return nil
}
