package handlers

import (
	"log/slog"

	"github.com/narender/common/globals"
	"github.com/narender/master-store/src/services"
)

type MasterStoreHandler struct {
	service services.MasterStoreService
	logger  *slog.Logger
}

func NewMasterStoreHandler(svc services.MasterStoreService) *MasterStoreHandler {
	return &MasterStoreHandler{
		service: svc,
		logger:  globals.Logger(),
	}
}
