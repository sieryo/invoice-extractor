package handler

import (
	"path/filepath"

	"github.com/gofiber/fiber/v2"

	appbuyer "github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
	"github.com/sieryo/invoice-extractor/internal/infra/storage"
)

type BuyerUploadHandler struct {
	registry *appbuyer.Registry
	parser   *parser.BuyerExcelParser
	store    *storage.BuyerCSVStore
	dataDir  string
}

func NewBuyerUploadHandler(
	registry *appbuyer.Registry,
	store *storage.BuyerCSVStore,
	dataDir string,
) *BuyerUploadHandler {
	return &BuyerUploadHandler{
		registry: registry,
		parser:   parser.NewBuyerExcelParser(),
		store:    store,
		dataDir:  dataDir,
	}
}

func (h *BuyerUploadHandler) IsLoaded(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"loaded": h.registry.IsLoaded(),
	})
}

func (h *BuyerUploadHandler) Handle(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "file required"})
	}

	tmpPath := filepath.Join(h.dataDir, "buyer_upload.xlsx")
	if err := c.SaveFile(file, tmpPath); err != nil {
		return err
	}

	buyers, err := h.parser.Parse(tmpPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": err.Error()})
	}

	if err := h.store.Save(buyers); err != nil {
		return err
	}

	return c.JSON(fiber.Map{
		"status": "ok",
		"count":  len(buyers),
	})
}
