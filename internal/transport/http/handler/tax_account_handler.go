package handler

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"

	appcashflow "github.com/sieryo/invoice-extractor/internal/app/cashflow"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
)

type TaxAccountHandler struct {
	service *appcashflow.TaxAccountService
}

func NewTaxAccountHandler(service *appcashflow.TaxAccountService) *TaxAccountHandler {
	return &TaxAccountHandler{service: service}
}

func (h *TaxAccountHandler) List(c *fiber.Ctx) error {
	accounts, err := h.service.List()
	if err != nil {
		var pathErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &pathErr) || os.IsNotExist(err) {
			return SendSuccess(c, fiber.StatusOK, fiber.Map{
				"accounts": []appcashflow.TaxAccount{},
				"count":    0,
				"status":   h.service.Status(),
				"version":  h.service.Spec().SchemaVersion,
			}, "tax account list retrieved")
		}
		return SendError(c, fiber.StatusBadRequest, "gagal memuat tax accounts")
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"accounts": accounts,
		"count":    len(accounts),
		"status":   h.service.Status(),
		"version":  h.service.Spec().SchemaVersion,
	}, "tax account list retrieved")
}

func (h *TaxAccountHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(), "tax account schema spec retrieved")
}

func (h *TaxAccountHandler) Status(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Status(), "tax account status retrieved")
}

func (h *TaxAccountHandler) Update(c *fiber.Ctx) error {
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
			message = "file tidak memenuhi format upload tax accounts"
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
		var schemaErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &schemaErr) {
			missing := append([]string(nil), schemaErr.MissingColumns...)
			sort.Strings(missing)
			required := requiredTaxAccountColumns(h.service.Spec())
			return SendError(
				c,
				fiber.StatusBadRequest,
				fmt.Sprintf(
					"schema tax accounts tidak sesuai. Kolom wajib: %s. Kolom hilang: %s",
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
	}, "tax account uploaded successfully")
}

func requiredTaxAccountColumns(spec parser.TaxAccountSchemaSpec) []string {
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
