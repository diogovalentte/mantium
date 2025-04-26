package suwayomi

import (
	"net/http"

	"github.com/diogovalentte/mantium/api/src/config"
)

type Suwayomi struct {
	c       *http.Client
	Address string
}

func (s *Suwayomi) Init() {
	s.c = &http.Client{}
	s.Address = config.GlobalConfigs.Suwayomi.Address
}
