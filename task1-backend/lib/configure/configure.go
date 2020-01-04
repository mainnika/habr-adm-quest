package configure

import (
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"

	"github.com/mainnika/habr-adm-quest/task1-backend/lib"
	"github.com/mainnika/habr-adm-quest/task1-backend/lib/env"
)

var Config lib.AppConfig

func init() {

	path := env.ConfigPath
	name := env.ConfigName

	if len(path) == 0 {
		path = lib.ConfPath
	}

	if len(name) == 0 {
		name = lib.ConfName
	}

	viper.AddConfigPath(path)
	viper.SetConfigName(name)
	viper.SetEnvPrefix(env.Prefix)
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	err, _ := viper.ReadInConfig(), viper.Unmarshal(&Config)
	if err != nil {
		logrus.Fatal(err)
	}
}
