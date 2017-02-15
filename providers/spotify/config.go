package spotify

import "github.com/spf13/viper"

const (
	vUsername = "spotify.username"
	vPassword = "spotify.password"
	vAPIKey   = "spotify.apikey"
)

type Configurer interface {
	Username() string
	Password() string
	APIKey() string
}

func init() {
	viper.BindEnv(vUsername, vPassword, vAPIKey)
}

type Config struct{}

func (c Config) Username() string {
	return viper.GetString(vUsername)
}

func (c Config) Password() string {
	return viper.GetString(vPassword)
}

func (c Config) APIKey() string {
	return viper.GetString(vAPIKey)
}

func NewConfig() Config {
	return Config{}
}
