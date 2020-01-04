package lib

var (
	ConfPath = "config"
	ConfName = "task1"
)

type AppConfig struct {
	HttpAPI struct {
		Base string `mapstructure:"base"`
		Addr string `mapstructure:"addr"`
	} `mapstructure:"httpApi"`
}
