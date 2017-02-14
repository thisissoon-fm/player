package web

import (
	"time"

	"github.com/spf13/viper"
)

const (
	vHost  = "websocket.host"
	vRetry = "websocket.retru"
)

func init() {
	viper.BindEnv(vHost)
	viper.SetDefault(vHost, "localhost:8000")
	viper.BindEnv(vRetry)
	viper.SetDefault(vRetry, "5s")
}

type Configurer interface {
	Host() string
	Retry() time.Duration
}

type Config struct{}

func (c Config) Host() string {
	return viper.GetString(vHost)
}

func (c Config) Retry() time.Duration {
	return viper.GetDuration(vRetry)
}

func NewConfig() Config {
	return Config{}
}
