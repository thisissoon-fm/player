package soundcloud

import "github.com/spf13/viper"

const (
	vClientID     = "soundcloud.client_id"
	vClientSecret = "soundcloud.client_secret"
	vAPIHost      = "soundcloud.api_host"
	vAPIScheme    = "soundcloud.api_scheme"
)

type Configurer interface {
	ClientID() string
	ClientSecret() string
	APIHost() string
	APIScheme() string
}

func init() {
	viper.SetDefault(vAPIHost, "api.soundcloud.com")
	viper.SetDefault(vAPIScheme, "https")
	viper.BindEnv(
		vClientID,
		vClientSecret,
		vAPIHost,
		vAPIScheme)
}

type Config struct{}

func (c Config) ClientID() string {
	return viper.GetString(vClientID)
}

func (c Config) ClientSecret() string {
	return viper.GetString(vClientSecret)
}

func (c Config) APIHost() string {
	return viper.GetString(vAPIHost)
}

func (c Config) APIScheme() string {
	return viper.GetString(vAPIScheme)
}

func NewConfig() Config {
	return Config{}
}
