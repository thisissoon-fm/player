// Logger Configuration
//
// Example TOML:
// [log]
// level = "debug"
// logfile = "/path/to/scoreboard.log
// format = "logstash"
//
// [logstash]
// type = "mylogstashtype"
//
// Environment Variables:
// SFMPLAYER_LOG_LEVEL = "info"
// SFMPLAYER_LOG_LOGFILE = "/path/to/scoreboard.log"
// SFMPLAYER_LOG_FORMAT = "logstash"
//
// CLI Flags:
// -l/--level info

package logger

import (
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

// Set logging configuration defaults
func init() {
	viper.BindEnv("LOG.LEVEL")
	viper.SetDefault("log.level", "info")
	viper.BindEnv("LOG.FORMAT")
	viper.SetDefault("log.format", "text")
	viper.BindEnv("LOG.LOGFILE")
	viper.SetDefault("log.logfile", "")
	viper.SetDefault("log.console_output", true)
	viper.SetDefault("logstash.type", "")
}

// Logger configuration interface
type Configurer interface {
	Level() string
	Format() string
	LogFile() string
	ConsoleOutput() bool
	LogstashType() string
}

// Allows us to bind a cli flag to a viper config option for log.level
func BindLogLevelFlag(flag *pflag.Flag) {
	viper.BindPFlag("log.level", flag)
}

// A simple type for accessing logging configuration
type Config struct{}

// Returns the logging verbosity level, binds to environment variable
func (c Config) Level() string {
	return viper.GetString("log.level")
}

// Returns absolute path to logfile
func (c Config) LogFile() string {
	return viper.GetString("log.logfile")
}

// Returns logging format to use
func (c Config) Format() string {
	return viper.GetString("log.format")
}

// Returns console log output bool
func (c Config) ConsoleOutput() bool {
	return viper.GetBool("log.console_output")
}

// Returns logstash format type
func (c Config) LogstashType() string {
	return viper.GetString("logstash.type")
}

// Constructs a Config
func NewConfig() Config {
	return Config{}
}
