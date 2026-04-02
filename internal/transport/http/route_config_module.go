package http

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/configmodule"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerConfigModuleRoutes(protected fiber.Router) {
	config := protected.Group("/config")
	pageService := configmodule.NewPageService(
		s.appCtx.SettingsService,
		s.appCtx.BuyerRegistryService,
		s.appCtx.TemplateRegistryService,
		s.appCtx.BukpotRequestConfigService,
		s.appCtx.BukpotActionProfileService,
		s.appCtx.CashflowProfileConfigService,
		s.appCtx.CashflowBillProfileService,
		s.appCtx.TaxAccountService,
		s.appCtx.CashflowBillCategoryService,
	)
	h := handler.NewConfigModuleHandler(s.appCtx.SettingsService, s.appCtx.ModuleActivationService, pageService)
	config.Get("/modules", h.List)
	config.Get("/modules/:moduleKey", h.Get)
	config.Get("/modules/:moduleKey/page", h.Page)
	config.Put("/modules/:moduleKey/blocks/:blockKey", h.UpdateBlock)
	config.Post("/modules/:moduleKey/blocks/:blockKey/upload", h.UploadBlock)
}
