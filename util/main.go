package util

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/jsmootiv/piq/pools"
)

type config struct {
	Workers []string     `json:"workers"`
	Pools   []pools.Pool `json:"pools"`
}

func OpenConfig(location string) (*config, error) {
	cfg := &config{}
	jsonFile, err := os.Open(location)

	if err != nil {
		currUser, err := user.Current()
		if err != nil {
			return cfg, nil
		}
		userCfg := currUser.HomeDir + "/.piq/config.json"
		if location == userCfg {
			return cfg, errors.New("No config found")
		}
		return OpenConfig(userCfg)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return cfg, nil
	}
	json.Unmarshal(byteValue, cfg)
	return cfg, nil
}
