package colossusconfig

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Colossus struct {
		Logging struct {
			StackDriver  bool `desc:"Send Logs to GCP Stackdriver"`
			StackDriver_ struct {
				UseLoggingAgent bool `desc:"Use the GCP Logging Agent to send logs"`
				UseGCE bool `desc:"Use the GCE Metadata to configure the logger"`
				UseApplicationDefaultCredentials bool `desc:"Use the ApplicationDefaultCredentials to send logs directly to GCP"`
			} `desc:"Stackdriver Logging Config" envconfig:"STACKDRIVER"`
			DisableConsole bool `desc:"Disable Console Logging"`
		} `desc:"Colossus Logging configuration"`
	} `desc:"The global Colossus configuration"`
	Google struct {
		Application struct {
			Credentials string
		}
	}
}

var DefaultConfig = &Config{}

func init()  {
	envconfig.MustProcess("", DefaultConfig)
}