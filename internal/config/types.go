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
	AdminKey        string `yaml:"adminKey"`
	EnableAuth      bool   `yaml:"enableAuth"`
	MaxRequestSize  int64  `yaml:"maxRequestSize"`  // in bytes
	RateLimitPerMin int    `yaml:"rateLimitPerMin"` // requests per minute
	RateLimitBurst  int    `yaml:"rateLimitBurst"`  // max burst size
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
	HTTP          TCPServer     `yaml:"http"`
	Log           Logging       `yaml:"log"`
	Redis         Redis         `yaml:"redis"`
	MQTT          MQTT          `yaml:"mqtt"`
	InternetCheck InternetCheck `yaml:"internetCheck"`
	Security      Security      `yaml:"security"`
	Sentry        Sentry        `yaml:"sentry"`
	OTel          OTel          `yaml:"otel"`

	DiscordWebhookError   DiscordWebhook `yaml:"discordWebhookError"`
	DiscordWebhookWarning DiscordWebhook `yaml:"discordWebhookWarning"`
	DiscordWebhookInfo    DiscordWebhook `yaml:"discordWebhookInfo"`
}
