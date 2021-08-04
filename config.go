package main

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
)

type cfgResolver struct {
	Type string   `koanf:"type"`
	Urls []string `koanf:"urls"`
}

type cfgCache struct {
	Cache    bool `koanf:"cache"`
	MaxItems int  `koanf:"max_items"`
}

type cfgLog struct {
	LogLevel   string `koanf:"log_level"`
	LogQueries bool   `koanf:"log_queries"`
}

type Config struct {
	BindAddress      string      `koanf:"bind_address"`
	BootstrapAddress string      `koanf:"bootstrap_address"`
	Cache            cfgCache    `koanf:"cache"`
	Resolver         cfgResolver `koanf:"resolver"`
	Log              cfgLog      `koanf:"log"`
}

func initConfig(cfgPath string) (Config, error) {
	k := koanf.New(".")

	// Set default values
	k.Load(confmap.Provider(map[string]interface{}{
		"cache.cache":       true,
		"bind_address":      "127.0.0.1:53",
		"bootstrap_address": "9.9.9.9:53",
		"resolver.type":     "doh",
	}, "."), nil)

	if err := k.Load(file.Provider(cfgPath), toml.Parser()); err != nil {
		return Config{}, err
	}

	cfg := Config{}

	if err := k.Unmarshal("", &cfg); err != nil {
		return Config{}, err
	}

	return cfg, nil
}
