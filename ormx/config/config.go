package config

import internal "go-ormx/ormx/internal/config"

// Re-export public API from internal/config so consumers outside the internal tree can use it

type Config = internal.Config
type DatabaseType = internal.DatabaseType

const (
	PostgreSQL DatabaseType = internal.PostgreSQL
	MySQL      DatabaseType = internal.MySQL
)

func DefaultConfig() *Config        { return internal.DefaultConfig() }
func LoadFromEnv() (*Config, error) { return internal.LoadFromEnv() }
