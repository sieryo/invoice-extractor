package http

import (
	"github.com/gofiber/fiber/v2"
	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerCashflowBillProfileConfigRoutes(protected fiber.Router) {
	if s.appCtx.CashflowBillProfileService == nil {
		return
	}

	payBillsHandler := handler.NewCashflowBillProfileConfigHandler(
		s.appCtx.CashflowBillProfileService,
		appcashflowbill.ProfileConfigPayBills,
	)
	receivePaymentsHandler := handler.NewCashflowBillProfileConfigHandler(
		s.appCtx.CashflowBillProfileService,
		appcashflowbill.ProfileConfigReceivePayments,
	)

	config := protected.Group("/config/profile/cashflow-bills")
	config.Get("/pay-bills", payBillsHandler.Get)
	config.Get("/pay-bills/spec", payBillsHandler.Spec)
	config.Get("/pay-bills/status", payBillsHandler.Status)
	config.Put("/pay-bills", payBillsHandler.Update)

	config.Get("/receive-payments", receivePaymentsHandler.Get)
	config.Get("/receive-payments/spec", receivePaymentsHandler.Spec)
	config.Get("/receive-payments/status", receivePaymentsHandler.Status)
	config.Put("/receive-payments", receivePaymentsHandler.Update)
}
