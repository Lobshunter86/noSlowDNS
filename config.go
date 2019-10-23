package DNS

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"time"
)

type Config struct {
	Net               string `json:"Net"`
	Addr              string `json:"Addr"`
	PrimaryUpstream   string `json:"PrimaryUpstream"`
	SecondaryUpstream string `json:"SecondaryUpstream"`

	ExpireTime  time.Duration `json:"ExpireTime"`
	CleanupTime time.Duration `json:"CleanupTime"`
}

func ReadConfig(path string) *Config {
	file, err := os.Open(path)
	must(err)
	cfgFile, err := ioutil.ReadAll(file)
	must(err)

	var cfg *Config
	err = json.Unmarshal(cfgFile, &cfg)
	must(err)

	return cfg
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
