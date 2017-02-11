package unix

import "github.com/spf13/viper"

const (
	vAddress = "unixsocket.address"
)

func init() {
	viper.BindEnv(vAddress)
	viper.SetDefault(vAddress, "/tmp/sfmplayer.sock")
}

type Configurer interface {
	Address() string
}

type Config struct{}

func (c Config) Address() string {
	return viper.GetString(vAddress)
}

func NewConfig() Config {
	return Config{}
}
