package config

import "time"

type Config struct {
	MaxPages       int
	WaitTimeMin    time.Duration
	WaitTimeMax    time.Duration
	PageDelay      time.Duration
	Headless       bool
	UserAgent      string
	Categories     []CategoryConfig
	CloudflareWait time.Duration
	RetryAttempts  int
}

type CategoryConfig struct {
	Name    string
	URL     string
	Filter  string
	Targets []string
}

func NewDefault() *Config {
	return &Config{
		MaxPages:       5,
		WaitTimeMin:    4 * time.Second,
		WaitTimeMax:    9 * time.Second,
		PageDelay:      10 * time.Second,
		Headless:       false,
		UserAgent:      "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
		CloudflareWait: 30 * time.Second,
		RetryAttempts:  3,
		Categories: []CategoryConfig{
			{
				Name:   "GPU",
				URL:    "https://www.pichau.com.br/hardware/placa-de-video",
				Filter: "placa",
			},
			{
				Name:   "CPU",
				URL:    "https://www.pichau.com.br/hardware/processadores",
				Filter: "processador",
			},
		},
	}
}
