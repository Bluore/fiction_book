package server

import (
	"context"

	"fiction-book/pkg/config"
)

type Handle struct {
	orderTempServer *OrderTempServer
	cfg             *config.Config
}

func NewHandle(orderTempServer *OrderTempServer, cfg *config.Config) *Handle {
	return &Handle{
		orderTempServer: orderTempServer,
		cfg:             cfg,
	}
}

func (h *Handle) Start(ctx context.Context) {
	go h.orderTempServer.OrderTempServerInit(ctx)
}
