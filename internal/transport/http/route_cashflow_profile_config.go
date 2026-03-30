package http

import (
	"github.com/gofiber/fiber/v2"
	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerCashflowProfileConfigRoutes(protected fiber.Router) {
	if s.appCtx.CashflowProfileConfigService == nil {
		return
	}

	spendHandler := handler.NewCashflowProfileConfigHandler(
		s.appCtx.CashflowProfileConfigService,
		appcashflow.ProfileConfigSpendMoney,
	)
	receiveHandler := handler.NewCashflowProfileConfigHandler(
		s.appCtx.CashflowProfileConfigService,
		appcashflow.ProfileConfigReceiveMoney,
	)

	config := protected.Group("/config/profile/cashflow")
	config.Get("/spend-money", spendHandler.Get)
	config.Get("/spend-money/spec", spendHandler.Spec)
	config.Get("/spend-money/status", spendHandler.Status)
	config.Put("/spend-money", spendHandler.Update)

	config.Get("/receive-money", receiveHandler.Get)
	config.Get("/receive-money/spec", receiveHandler.Spec)
	config.Get("/receive-money/status", receiveHandler.Status)
	config.Put("/receive-money", receiveHandler.Update)
}
