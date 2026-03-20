package handler

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
)

type BuyerRegistryHandler struct {
	service *buyer.BuyerRegistryService
}

func NewBuyerRegistryHandler(service *buyer.BuyerRegistryService) *BuyerRegistryHandler {
	return &BuyerRegistryHandler{
		service: service,
	}
}

func (h *BuyerRegistryHandler) List(c *fiber.Ctx) error {
	buyers := h.service.List()
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"buyers":  buyers,
		"count":   len(buyers),
		"status":  h.service.Status(),
		"version": h.service.Spec().SchemaVersion,
	}, "buyer list retrieved")
}

func (h *BuyerRegistryHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(), "buyer schema spec retrieved")
}

func (h *BuyerRegistryHandler) Update(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "file required")
	}
	if file.Size <= 0 {
		return SendError(c, fiber.StatusBadRequest, "file kosong atau tidak valid")
	}
	if ok, reason := h.service.IsAcceptedUpload(file.Filename, file.Size); !ok {
		spec := h.service.Spec()
		allowed := strings.Join(spec.Upload.AcceptedExtensions, ", ")
		limit := spec.Upload.MaxFileSizeMB
		message := strings.TrimSpace(reason)
		if message == "" {
			message = "file tidak memenuhi format upload buyer registry"
		}
		return SendError(
			c,
			fiber.StatusBadRequest,
			fmt.Sprintf("%s (format: %s, maksimal: %dMB)", message, allowed, limit),
		)
	}

	tmpPath := h.service.TempFilePath()
	if err := c.SaveFile(file, tmpPath); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to save file")
	}

	count, issues, err := h.service.Update(tmpPath)
	if err != nil {
		var schemaErr *parser.BuyerSchemaMismatchError
		if errors.As(err, &schemaErr) {
			missing := append([]string(nil), schemaErr.MissingColumns...)
			sort.Strings(missing)
			required := requiredBuyerColumns(h.service.Spec())
			return SendError(
				c,
				fiber.StatusBadRequest,
				fmt.Sprintf(
					"schema buyer registry tidak sesuai. Kolom wajib: %s. Kolom hilang: %s",
					strings.Join(required, ", "),
					strings.Join(missing, ", "),
				),
			)
		}
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"count":  count,
		"issues": sanitizeValidationIssues(issues),
		"status": h.service.Status(),
	}, "buyer data uploaded successfully")
}

func (h *BuyerRegistryHandler) IsLoaded(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Status(), "buyer data status retrieved")
}

func sanitizeValidationIssues(issues []parser.ValidationIssue) []parser.ValidationIssue {
	if len(issues) == 0 {
		return nil
	}
	safe := make([]parser.ValidationIssue, 0, len(issues))
	for _, issue := range issues {
		safe = append(safe, parser.ValidationIssue{
			Row:     issue.Row,
			Field:   issue.Field,
			Message: issue.Message,
		})
	}
	return safe
}

func requiredBuyerColumns(spec parser.BuyerRegistrySchemaSpec) []string {
	columns := make([]string, 0, len(spec.Columns))
	for _, column := range spec.Columns {
		if !column.Required {
			continue
		}
		label := strings.TrimSpace(column.Header)
		if label == "" {
			continue
		}
		columns = append(columns, label)
	}
	sort.Strings(columns)
	return columns
}
