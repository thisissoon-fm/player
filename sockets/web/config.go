package web

import (
	"time"

	"github.com/spf13/viper"
)

const (
	vHost     = "websocket.host"
	vRetry    = "websocket.retry"
	vUsername = "websocket.username"
	vPassword = "websocket.password"
)

func init() {
	viper.SetDefault(vHost, "localhost:8000")
	viper.SetDefault(vRetry, "5s")
	viper.BindEnv(vHost, vRetry, vUsername, vPassword)
}

type Configurer interface {
	Host() string
	Retry() time.Duration
	Username() string
	Password() string
}

type Config struct{}

func (c Config) Host() string {
	return viper.GetString(vHost)
}

func (c Config) Retry() time.Duration {
	return viper.GetDuration(vRetry)
}

func (c Config) Username() string {
	return viper.GetString(vUsername)
}

func (c Config) Password() string {
	return viper.GetString(vPassword)
}

func NewConfig() Config {
	return Config{}
}
