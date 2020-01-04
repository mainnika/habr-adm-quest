package lib

var (
	ConfPath = "config"
	ConfName = "task2"
)

type AppConfig struct {
	HttpAPI struct {
		Base string `mapstructure:"base"`
		Addr string `mapstructure:"addr"`
	} `mapstructure:"httpApi"`
	Redis struct {
		Addr     string `mapstructure:"addr"`
		ScoreKey string `mapstructure:"scoreKey"`
		WinnersKey string `mapstructure:"winnersKey"`
	} `mapstructure:"redis"`
}
