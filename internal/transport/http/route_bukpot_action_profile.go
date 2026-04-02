package http

import (
	"github.com/gofiber/fiber/v2"
	appbukpot "github.com/sieryo/invoice-extractor/internal/app/bukpot"
	"github.com/sieryo/invoice-extractor/internal/transport/http/handler"
)

func (s *Server) registerBukpotActionProfileRoutes(protected fiber.Router) {
	if s.appCtx.BukpotActionProfileService == nil {
		return
	}

	config := protected.Group("/config/profile/bukpot-actions")

	for _, key := range []appbukpot.ActionProfileKey{
		appbukpot.ActionProfileBPPURenameBukpot,
		appbukpot.ActionProfileBP21RenameBukpot,
		appbukpot.ActionProfileBPA1RenameBukpot,
		appbukpot.ActionProfileBPPURenameByCategory,
		appbukpot.ActionProfileBP21RenameByCategory,
	} {
		h := handler.NewBukpotActionProfileHandler(s.appCtx.BukpotActionProfileService, key)
		base := "/" + string(key)
		config.Get(base, h.Get)
		config.Get(base+"/spec", h.Spec)
		config.Get(base+"/status", h.Status)
		config.Put(base, h.Update)
	}
}
