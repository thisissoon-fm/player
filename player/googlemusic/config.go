package googlemusic

import "github.com/spf13/viper"

const (
	vUsername = "googlemusic.username"
	vPassword = "googlemusic.password"
)

type Configurer interface {
	Username() string
	Password() string
}

func init() {
	viper.BindEnv(vUsername, vPassword)
}

type Config struct{}

func (c Config) Username() string {
	return viper.GetString(vUsername)
}

func (c Config) Password() string {
	return viper.GetString(vPassword)
}

func NewConfig() Config {
	return Config{}
}
