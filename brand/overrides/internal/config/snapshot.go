package config

import "github.com/spf13/viper"

// RuntimeConfig creates an in-memory snapshot; it never loads root's profile.
func RuntimeConfig(logFile string) *Config {
	v := viper.New()
	return &Config{v: v, LogFile: logFile, LogLevel: "info", DisableCompanionMode: true, DisableUpdateCheck: true}
}
