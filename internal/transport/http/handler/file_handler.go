package handler

import (
	"io"

	"github.com/gofiber/fiber/v2"
	appFile "github.com/sieryo/invoice-extractor/internal/app/file"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type FileHandler struct {
	fileService *appFile.FileService
}

func NewFileHandler(fileService *appFile.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
	}
}

func (h *FileHandler) Upload(c *fiber.Ctx) error {
	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection_id is required")
	}

	form, err := c.MultipartForm()
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid multipart form")
	}

	files := form.File["files"]
	if len(files) == 0 {
		return SendError(c, fiber.StatusBadRequest, "no files uploaded")
	}

	ctx := c.Context()

	var inputs []file.UploadInput

	for _, f := range files {
		src, err := f.Open()
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "failed to open file")
		}

		data, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "failed to read file")
		}

		inputs = append(inputs, file.UploadInput{
			Name: f.Filename,
			Data: data,
		})
	}

	uploaded, err := h.fileService.UploadFiles(ctx, collectionID, inputs)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, uploaded, "files uploaded")
}

func (h *FileHandler) GetFileObjectByID(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")

	if id == "" {
		return SendError(c, fiber.StatusBadRequest, "id is required")
	}

	f, err := h.fileService.GetByID(ctx, id)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, f, "file retrieved successfully")
}

func (h *FileHandler) ListByCollection(c *fiber.Ctx) error {
	ctx := c.Context()
	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collectionId is required")
	}

	files, err := h.fileService.ListByCollection(ctx, collectionID)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	if len(files) == 0 {
		return SendSuccess(c, fiber.StatusOK, []file.FileObject{}, "files retrieved successfully")
	}

	return SendSuccess(c, fiber.StatusOK, files, "files retrieved successfully")
}

func (h *FileHandler) DeleteFile(c *fiber.Ctx) error {
	ctx := c.Context()
	id := c.Params("id")
	if id == "" {
		return SendError(c, fiber.StatusBadRequest, "id is required")
	}

	if err := h.fileService.Delete(ctx, id); err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, nil, "file deleted successfully")
}

func (h *FileHandler) DeleteFilesBulk(c *fiber.Ctx) error {
	ctx := c.Context()
	var req struct {
		IDs []string `json:"ids"`
	}
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	if len(req.IDs) == 0 {
		return SendError(c, fiber.StatusBadRequest, "ids are required")
	}

	if err := h.fileService.DeleteBulk(ctx, req.IDs); err != nil {
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}
	return SendSuccess(c, fiber.StatusOK, nil, "files deleted successfully")
}
