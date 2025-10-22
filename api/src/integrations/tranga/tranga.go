package tranga

import (
	"net/http"

	"github.com/mendoncart/mantium/api/src/config"
)

type Tranga struct {
	c               *http.Client
	Address         string
	DefaultInterval string
}

func (t *Tranga) Init() {
	t.c = &http.Client{}
	t.Address = config.GlobalConfigs.Tranga.Address
	t.DefaultInterval = config.GlobalConfigs.Tranga.DefaultInterval
}
