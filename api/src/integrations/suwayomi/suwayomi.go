package suwayomi

import (
	"net/http"

	"github.com/diogovalentte/mantium/api/src/config"
	"github.com/diogovalentte/mantium/api/src/util"
)

type Suwayomi struct {
	c        *http.Client
	Address  string
	Username string
	Password string
}

func (s *Suwayomi) Init() {
	s.c = &http.Client{Timeout: util.ExternalRequestTimeout}
	s.Address = config.GlobalConfigs.Suwayomi.Address
	s.Username = config.GlobalConfigs.Suwayomi.Username
	s.Password = config.GlobalConfigs.Suwayomi.Password
}
