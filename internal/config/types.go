package config

import (
	"fmt"
)

type TCPServer struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

func (t TCPServer) Addr() string {
	return fmt.Sprintf("%s:%s", t.Host, t.Port)
}

type Logging struct {
	Level          string `yaml:"level"`
	Type           string `yaml:"type"`
	LogFileEnabled bool   `yaml:"logFileEnabled"`
	LogFilePath    string `yaml:"logFilePath"`
}

type Redis struct {
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

func (r Redis) Addr() string {
	return fmt.Sprintf("%s:%s", r.Host, r.Port)
}

type MQTT struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Host     string `yaml:"host"`
	Port     string `yaml:"port"`
}

func (m MQTT) BrokerURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%s/", m.User, m.Password, m.Host, m.Port)
}

type Security struct {
	BearerToken     string `yaml:"bearerToken"`
	MCPToken        string `yaml:"mcpToken"`
	MaxRequestSize  int64  `yaml:"maxRequestSize"`  // in bytes
	RateLimitPerMin int    `yaml:"rateLimitPerMin"` // requests per minute
	RateLimitBurst  int    `yaml:"rateLimitBurst"`  // max burst size
}

func (s Security) ValidateAuthTokens() error {
	if s.BearerToken == "" {
		return fmt.Errorf("security.bearerToken must be set")
	}
	if s.MCPToken == "" {
		return fmt.Errorf("security.mcpToken must be set")
	}
	return nil
}

func (s Security) GetMaxRequestSize() int64 {
	if s.MaxRequestSize == 0 {
		return 1024 * 1024 // 1MB
	}
	return s.MaxRequestSize
}

func (s Security) GetRateLimitPerMin() int {
	if s.RateLimitPerMin == 0 {
		return 60
	}
	return s.RateLimitPerMin
}

func (s Security) GetRateLimitBurst() int {
	if s.RateLimitBurst == 0 {
		return 10
	}
	return s.RateLimitBurst
}

type Sentry struct {
	DSN         string  `yaml:"dsn"`
	Environment string  `yaml:"environment"`
	SampleRate  float64 `yaml:"sampleRate"`
}

type OTel struct {
	Endpoint    string `yaml:"endpoint"`
	ServiceName string `yaml:"serviceName"`
	Enabled     bool   `yaml:"enabled"`
}

type Config struct {
	HTTP           TCPServer `yaml:"http"`
	MCP            TCPServer `yaml:"mcp"`
	Log            Logging   `yaml:"log"`
	Redis          Redis     `yaml:"redis"`
	MQTT           MQTT      `yaml:"mqtt"`
	Security       Security  `yaml:"security"`
	Sentry         Sentry    `yaml:"sentry"`
	OTel           OTel      `yaml:"otel"`
	RedisKeyPrefix string    `yaml:"redisKeyPrefix"`
}

func (c Config) GetRedisKeyPrefix() string {
	if c.RedisKeyPrefix == "" {
		return "hmauto"
	}
	return c.RedisKeyPrefix
}
