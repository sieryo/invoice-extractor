package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/collection"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
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
	Name string `json:"name"`
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

	newID := uuid.NewString()

	coll, err := h.collectionService.Create(ctx, newID, req.Name, userID)
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
		if err == dcollection.ErrCollectionNotFound {
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

	if len(collections) == 0 {
		return SendSuccess(c, fiber.StatusOK, []dcollection.Collection{}, "collections retrieved successfully")
	}

	return SendSuccess(c, fiber.StatusOK, collections, "collections retrieved successfully")
}
