package config

import (
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Configuration defaults
func init() {
	viper.SetTypeByDefaultValue(true)
	viper.SetConfigType("toml")
	viper.SetConfigName("config")
	viper.AddConfigPath("/etc/sfm/player")
	viper.AddConfigPath("$HOME/.config/sfm/player")
	viper.SetEnvPrefix("SFM")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
}

// Read configuration
func Read(path string) error {
	if _, err := os.Stat(path); err != nil {
		viper.SetConfigFile(path)
	}
	return viper.ReadInConfig()
}
