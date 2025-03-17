package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 应用配置结构
type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Proxy struct {
		ConnectTimeout   int   `yaml:"connectTimeout"`
		TransferTimeout  int   `yaml:"transferTimeout"`
		BufferSize       int   `yaml:"bufferSize"`
		ChunkedThreshold int64 `yaml:"chunkedThreshold"`
	} `yaml:"proxy"`

	Security struct {
		RateLimiting struct {
			Enabled           bool `yaml:"enabled"`
			RequestsPerMinute int  `yaml:"requestsPerMinute"`
		} `yaml:"rateLimiting"`
		PrivateIPBlocking bool `yaml:"privateIPBlocking"`
	} `yaml:"security"`

	Headers struct {
		RemoveProxyHeaders     bool   `yaml:"removeProxyHeaders"`
		RemoveSensitiveHeaders bool   `yaml:"removeSensitiveHeaders"`
		NodeID                 string `yaml:"nodeId"`
	} `yaml:"headers"`

	Logging struct {
		Level  string `yaml:"level"`
		Format string `yaml:"format"`
	} `yaml:"logging"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	cfg := &Config{}

	// 服务器配置
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 8080

	// 代理配置
	cfg.Proxy.ConnectTimeout = 5
	cfg.Proxy.TransferTimeout = 300
	cfg.Proxy.BufferSize = 32 * 1024
	cfg.Proxy.ChunkedThreshold = 100 * 1024 * 1024 // 100MB

	// 安全配置
	cfg.Security.RateLimiting.Enabled = true
	cfg.Security.RateLimiting.RequestsPerMinute = 60
	cfg.Security.PrivateIPBlocking = true

	// 头部配置
	cfg.Headers.RemoveProxyHeaders = true
	cfg.Headers.RemoveSensitiveHeaders = true
	cfg.Headers.NodeID = "node1"

	// 日志配置
	cfg.Logging.Level = "info"
	cfg.Logging.Format = "text"

	return cfg
}

// LoadConfig 从文件加载配置
func LoadConfig(filename string) (*Config, error) {
	// 使用默认配置
	config := DefaultConfig()

	// 如果配置文件不存在，创建默认配置文件
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// 确保目录存在
		dir := filepath.Dir(filename)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建配置目录失败: %v", err)
		}

		// 保存默认配置
		if err := SaveConfig(config, filename); err != nil {
			return nil, fmt.Errorf("保存默认配置失败: %v", err)
		}

		return config, nil
	}

	// 读取配置文件
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %v", err)
	}

	// 解析YAML
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return config, nil
}

// SaveConfig 保存配置到文件
func SaveConfig(config *Config, filename string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %v", err)
	}

	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %v", err)
	}

	return nil
}
