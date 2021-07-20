package main

import (
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/toml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
)

type cfgResolver struct {
	Urls []string `koanf:"urls"`
}

type Config struct {
	BindAddress string      `koanf:"bind_address"`
	Cache       bool        `koanf:"cache"`
	LogLevel    string      `koanf:"log_level"`
	LogQueries  bool        `koanf:"log_queries"`
	Resolver    cfgResolver `koanf:"resolver"`
}

func initConfig(cfgPath string) (Config, error) {
	k := koanf.New(".")

	// Set default values
	k.Load(confmap.Provider(map[string]interface{}{
		"cache":        true,
		"bind_address": "127.0.0.1:53",
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
