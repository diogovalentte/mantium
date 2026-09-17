package kaizoku

import (
	"net/http"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

type Kaizoku struct {
	Address         string
	DefaultInterval string
	c               *http.Client
}

func (k *Kaizoku) Init() {
	k.Address = config.GlobalConfigs.Kaizoku.Address
	k.DefaultInterval = config.GlobalConfigs.Kaizoku.DefaultInterval
	k.c = &http.Client{Timeout: util.ExternalRequestTimeout}
}
