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

type SQL struct {
	User        string `yaml:"user"`
	Password    string `yaml:"password"`
	Host        string `yaml:"host"`
	Name        string `yaml:"name"`
	Port        string `yaml:"port"`
	MaxIdleConn int    `yaml:"maxIdleConn"`
	MaxOpenConn int    `yaml:"maxOpenConn"`
	SSLMode     string `yaml:"sslMode"` // disable, require, verify-ca, verify-full
}

func (s SQL) DatabaseUrl() string {
	sslMode := s.SSLMode
	if sslMode == "" {
		sslMode = "require" // Default to require for security
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		s.User, s.Password, s.Host, s.Port, s.Name, sslMode)
}

func (s SQL) DataSourceName() string {
	sslMode := s.SSLMode
	if sslMode == "" {
		sslMode = "require" // Default to require for security
	}
	return fmt.Sprintf("user=%s password=%s host=%s port=%s dbname=%s sslmode=%s",
		s.User, s.Password, s.Host, s.Port, s.Name, sslMode)
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

type InternetCheck struct {
	CheckAddress string `yaml:"checkAddress"`
	ModemAddress string `yaml:"modemAddress"`
	Interval     string `yaml:"interval"`
	SwitchKey    string `yaml:"switchKey"`
}

type DiscordWebhook struct {
	URL string `yaml:"url"`
}

func (d DiscordWebhook) WebhookUrl() string {
	return d.URL
}

type Security struct {
	APIKeys          []string `yaml:"apiKeys"`
	EnableAuth       bool     `yaml:"enableAuth"`
	MaxRequestSize   int64    `yaml:"maxRequestSize"`   // in bytes
	RateLimitPerMin  int      `yaml:"rateLimitPerMin"`  // requests per minute
	RateLimitBurst   int      `yaml:"rateLimitBurst"`   // max burst size
}

type Config struct {
	HTTP          TCPServer     `yaml:"http"`
	Log           Logging       `yaml:"log"`
	DB            SQL           `yaml:"db"`
	MQTT          MQTT          `yaml:"mqtt"`
	InternetCheck InternetCheck `yaml:"internetCheck"`
	Security      Security      `yaml:"security"`

	DiscordWebhookError   DiscordWebhook `yaml:"discordWebhookError"`
	DiscordWebhookWarning DiscordWebhook `yaml:"discordWebhookWarning"`
	DiscordWebhookInfo    DiscordWebhook `yaml:"discordWebhookInfo"`
}
