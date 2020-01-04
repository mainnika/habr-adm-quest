package env

import (
	"fmt"
	"os"
)

var (
	Prefix        = "CFG"
	IsDevelopment = len(os.Getenv("DEBUG")) > 0
	ConfigPath    = os.Getenv(fmt.Sprintf("%s_PATH", Prefix))
	ConfigName    = os.Getenv(fmt.Sprintf("%s_NAME", Prefix))
)
