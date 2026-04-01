package handler

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/gofiber/fiber/v2"
	appcashflowbill "github.com/sieryo/invoice-extractor/internal/app/cashflowbill"
	"github.com/sieryo/invoice-extractor/internal/infra/parser"
)

type CategoryAccountHandler struct {
	service *appcashflowbill.CategoryAccountService
}

func NewCategoryAccountHandler(service *appcashflowbill.CategoryAccountService) *CategoryAccountHandler {
	return &CategoryAccountHandler{service: service}
}

func (h *CategoryAccountHandler) List(c *fiber.Ctx) error {
	profileID, _ := c.Locals("userId").(string)
	accounts, err := h.service.List(profileID)
	if err != nil {
		var pathErr *parser.TaxAccountSchemaMismatchError
		if errors.As(err, &pathErr) || os.IsNotExist(err) {
			return SendSuccess(c, fiber.StatusOK, fiber.Map{
				"accounts": []appcashflowbill.CategoryAccount{},
				"count":    0,
				"status":   h.service.Status(profileID),
				"version":  h.service.Spec().SchemaVersion,
			}, "category account list retrieved")
		}
		return SendError(c, fiber.StatusBadRequest, "gagal memuat category accounts")
	}
	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"accounts": accounts,
		"count":    len(accounts),
		"status":   h.service.Status(profileID),
		"version":  h.service.Spec().SchemaVersion,
	}, "category account list retrieved")
}

func (h *CategoryAccountHandler) Spec(c *fiber.Ctx) error {
	return SendSuccess(c, fiber.StatusOK, h.service.Spec(), "category account schema spec retrieved")
}

func (h *CategoryAccountHandler) Status(c *fiber.Ctx) error {
	profileID, _ := c.Locals("userId").(string)
	return SendSuccess(c, fiber.StatusOK, h.service.Status(profileID), "category account status retrieved")
}

func (h *CategoryAccountHandler) Update(c *fiber.Ctx) error {
	profileID, _ := c.Locals("userId").(string)
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
			message = "file tidak memenuhi format upload category accounts"
		}
		return SendError(
			c,
			fiber.StatusBadRequest,
			fmt.Sprintf("%s (format: %s, maksimal: %dMB)", message, allowed, limit),
		)
	}

	tmpPath := h.service.TempFilePath(profileID)
	if err := c.SaveFile(file, tmpPath); err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to save file")
	}

	count, issues, err := h.service.Update(profileID, tmpPath)
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
					"schema category accounts tidak sesuai. Kolom wajib: %s. Kolom hilang: %s",
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
		"status": h.service.Status(profileID),
	}, "category account uploaded successfully")
}
