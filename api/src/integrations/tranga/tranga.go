package tranga

import (
	"net/http"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

type Tranga struct {
	c               *http.Client
	Address         string
	DefaultInterval string
}

func (t *Tranga) Init() {
	t.c = &http.Client{Timeout: util.ExternalRequestTimeout}
	t.Address = config.GlobalConfigs.Tranga.Address
	t.DefaultInterval = config.GlobalConfigs.Tranga.DefaultInterval
}
