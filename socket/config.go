package socket

import "github.com/spf13/viper"

const (
	vAddress = "socket.address"
)

func init() {
	viper.BindEnv(vAddress)
	viper.SetDefault(vAddress, "/tmp/sfm.player.sock")
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
