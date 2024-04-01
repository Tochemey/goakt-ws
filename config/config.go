package config

import "github.com/caarlos0/env/v10"

// Config defines the server config
type Config struct {
	Port int32 `env:"PORT"`
}

// getConfig returns the server config
func GetConfig() (*Config, error) {
	// load the configuration
	config := &Config{}
	opts := env.Options{RequiredIfNoDef: true, UseFieldNameByDefault: false}
	if err := env.ParseWithOptions(config, opts); err != nil {
		return nil, err
	}
	return config, nil
}
