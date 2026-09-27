package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config map[string]string

func LoadEnv(keys ...string) Config {
	cfg := Config{}
	for _, key := range keys {
		if val, ok := os.LookupEnv(key); ok {
			cfg[key] = val
		}
	}
	return cfg
}

func LoadEnvFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := Config{}
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		val = strings.Trim(val, `"'`)
		cfg[key] = val
		_ = os.Setenv(key, val)
	}
	return cfg, nil
}

func LoadJSONFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	raw := map[string]any{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	cfg := Config{}
	flatten("", raw, cfg)
	return cfg, nil
}

func flatten(prefix string, in map[string]any, out Config) {
	for k, v := range in {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch t := v.(type) {
		case map[string]any:
			flatten(key, t, out)
		default:
			out[key] = fmt.Sprint(t)
		}
	}
}

func (c Config) Get(key, fallback string) string {
	if v, ok := c[key]; ok && v != "" {
		return v
	}
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func (c Config) Int(key string, fallback int) int {
	v := c.Get(key, "")
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func (c Config) Bool(key string, fallback bool) bool {
	v := strings.ToLower(c.Get(key, ""))
	switch v {
	case "1", "true", "yes", "on":
		return true
	case "0", "false", "no", "off":
		return false
	default:
		return fallback
	}
}

func (c Config) Duration(key string, fallback time.Duration) time.Duration {
	v := c.Get(key, "")
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}

func MustLoad(path string) Config {
	cfg, err := LoadEnvFile(path)
	if err != nil {
		panic(err)
	}
	return cfg
}
