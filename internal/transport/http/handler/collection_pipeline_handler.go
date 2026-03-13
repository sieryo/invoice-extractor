package handler

import (
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type CollectionPipelineHandler struct {
	ingestService *ingest.IngestService
}

func NewCollectionPipelineHandler(ingestService *ingest.IngestService) *CollectionPipelineHandler {
	return &CollectionPipelineHandler{
		ingestService: ingestService,
	}
}

func (h *CollectionPipelineHandler) ListDocuments(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	limit, err := strconv.Atoi(c.Query("limit", "100"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "limit must be a number")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "offset must be a number")
	}

	docs, err := h.ingestService.ListDocuments(
		ctx,
		userID,
		collectionID,
		c.Query("status"),
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, dcollection.ErrCollectionNotFound) {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, docs, "documents retrieved")
}

func (h *CollectionPipelineHandler) ListHistory(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "limit must be a number")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "offset must be a number")
	}

	history, err := h.ingestService.ListHistory(ctx, userID, collectionID, limit, offset)
	if err != nil {
		if errors.Is(err, dcollection.ErrCollectionNotFound) {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, history, "history retrieved")
}

func (h *CollectionPipelineHandler) ListHistoryItems(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}
	historyID := c.Params("historyId")
	if historyID == "" {
		return SendError(c, fiber.StatusBadRequest, "history id is required")
	}

	limit, err := strconv.Atoi(c.Query("limit", "100"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "limit must be a number")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "offset must be a number")
	}

	items, err := h.ingestService.ListHistoryItems(
		ctx,
		userID,
		collectionID,
		historyID,
		c.Query("status"),
		limit,
		offset,
	)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, ingest.ErrHistoryNotFound):
			return SendError(c, fiber.StatusNotFound, "history not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, items, "history items retrieved")
}
