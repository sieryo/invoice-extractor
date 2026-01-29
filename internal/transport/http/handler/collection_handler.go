package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/collection"
)

type CollectionHandler struct {
	collectionService *collection.CollectionService
}

func NewCollectionHandler(collectionService *collection.CollectionService) *CollectionHandler {
	return &CollectionHandler{
		collectionService: collectionService,
	}
}

type CreateCollectionRequest struct {
	ID string `json:"id"`
}

func (h *CollectionHandler) CreateCollection(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	var req CreateCollectionRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if req.ID == "" {
		return SendError(c, fiber.StatusBadRequest, "id is required")
	}

	coll, err := h.collectionService.Create(ctx, req.ID, userID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusCreated, coll, "collection created successfully")
}

func (h *CollectionHandler) GetCollectionByID(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		return SendError(c, fiber.StatusBadRequest, "id is required")
	}

	coll, err := h.collectionService.GetByID(ctx, id)
	if err != nil {
		if err == collection.ErrCollectionNotFound {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, coll, "collection retrieved successfully")
}

func (h *CollectionHandler) ListUserCollections(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collections, err := h.collectionService.ListByUser(ctx, userID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, collections, "collections retrieved successfully")
}
