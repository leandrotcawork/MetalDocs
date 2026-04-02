package config

import "os"

type GotenbergConfig struct {
	Enabled bool
	URL     string
}

func LoadGotenbergConfig() GotenbergConfig {
	url := os.Getenv("METALDOCS_GOTENBERG_URL")
	if url == "" {
		return GotenbergConfig{}
	}
	return GotenbergConfig{
		Enabled: true,
		URL:     url,
	}
}
