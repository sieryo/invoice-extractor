package handler

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/collection"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type CollectionHandler struct {
	collectionService *collection.CollectionService
}

type CollectionPathNode struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	NodeType string  `json:"nodeType"`
	ParentID *string `json:"parentId,omitempty"`
}

func NewCollectionHandler(collectionService *collection.CollectionService) *CollectionHandler {
	return &CollectionHandler{
		collectionService: collectionService,
	}
}

type CreateCollectionRequest struct {
	Name               string  `json:"name"`
	NodeType           string  `json:"nodeType,omitempty"`
	ParentID           *string `json:"parentId,omitempty"`
	DocumentType       *string `json:"documentType,omitempty"`
	LegacyNodeType     string  `json:"node_type,omitempty"`
	LegacyParentID     *string `json:"parent_id,omitempty"`
	LegacyDocumentType *string `json:"document_type,omitempty"`
}

func (h *CollectionHandler) CreateCollection(c *fiber.Ctx) error {
	return h.createNode(c, true)
}

func (h *CollectionHandler) CreateNode(c *fiber.Ctx) error {
	return h.createNode(c, false)
}

func (h *CollectionHandler) createNode(c *fiber.Ctx, legacyDefault bool) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	var req CreateCollectionRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		return SendError(c, fiber.StatusBadRequest, "name is required")
	}

	nodeType := req.NodeType
	if strings.TrimSpace(nodeType) == "" {
		nodeType = req.LegacyNodeType
	}
	nodeTypeRaw := strings.ToLower(strings.TrimSpace(nodeType))
	if nodeTypeRaw == "" {
		if legacyDefault {
			nodeTypeRaw = string(dcollection.NodeTypeCollection)
		} else {
			nodeTypeRaw = string(dcollection.NodeTypeCollection)
		}
	}

	newID := uuid.NewString()

	var (
		coll *dcollection.Collection
		err  error
	)

	parentID := req.ParentID
	if parentID == nil {
		parentID = req.LegacyParentID
	}

	switch dcollection.NodeType(nodeTypeRaw) {
	case dcollection.NodeTypeFolder:
		coll, err = h.collectionService.CreateFolder(ctx, newID, name, userID, parentID)
	case dcollection.NodeTypeCollection:
		docType := dcollection.DocumentTypePDFInvoice
		documentType := req.DocumentType
		if documentType == nil {
			documentType = req.LegacyDocumentType
		}
		if documentType != nil && strings.TrimSpace(*documentType) != "" {
			docType = dcollection.DocumentType(strings.ToLower(strings.TrimSpace(*documentType)))
		}
		coll, err = h.collectionService.CreateTypedCollection(ctx, newID, name, userID, parentID, docType)
	default:
		return SendError(c, fiber.StatusBadRequest, "invalid node_type")
	}

	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrInvalidDocumentType):
			return SendError(c, fiber.StatusBadRequest, "invalid document_type")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
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

func (h *CollectionHandler) ListChildren(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	var parentID *string
	if rawParent := strings.TrimSpace(c.Query("parent_id")); rawParent != "" {
		parentID = &rawParent
	}

	collections, err := h.collectionService.ListChildren(ctx, userID, parentID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	if len(collections) == 0 {
		return SendSuccess(c, fiber.StatusOK, []dcollection.Collection{}, "collections retrieved successfully")
	}

	return SendSuccess(c, fiber.StatusOK, collections, "collections retrieved successfully")
}

func (h *CollectionHandler) GetCollectionPath(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	id := strings.TrimSpace(c.Params("id"))
	if id == "" {
		return SendError(c, fiber.StatusBadRequest, "id is required")
	}

	path, err := h.collectionService.GetPath(ctx, userID, id)
	if err != nil {
		if err == dcollection.ErrCollectionNotFound {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	nodes := make([]CollectionPathNode, 0, len(path))
	for _, item := range path {
		if item == nil {
			continue
		}
		nodes = append(nodes, CollectionPathNode{
			ID:       item.ID,
			Name:     item.Name,
			NodeType: string(item.NodeType),
			ParentID: item.Parent,
		})
	}

	return SendSuccess(c, fiber.StatusOK, nodes, "collection path retrieved successfully")
}

func (h *CollectionHandler) DeleteCollection(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")
	if id == "" {
		return SendError(c, fiber.StatusBadRequest, "id is required")
	}

	if err := h.collectionService.Delete(ctx, id); err != nil {
		if err == dcollection.ErrCollectionNotFound {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, nil, "collection deleted successfully")
}
