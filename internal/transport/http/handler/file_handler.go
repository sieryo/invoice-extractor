package handler

import (
	"io"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type FileHandler struct {
	fileStore file.FileStore
}

func NewFileHandler(fileStore file.FileStore) *FileHandler {
	return &FileHandler{
		fileStore: fileStore,
	}
}

// Definisi fungsi untuk buat collection

func (h *FileHandler) Upload(c *fiber.Ctx) error {
	form, err := c.MultipartForm()
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid multipart form")
	}

	files := form.File["files"]
	if len(files) == 0 {
		return SendError(c, fiber.StatusBadRequest, "no files uploaded")
	}

	ctx := c.Context()
	var uploaded []file.FileObject

	for _, file := range files {
		src, err := file.Open()
		if err != nil {
			return SendError(c, 500, "failed to open file")
		}
		defer src.Close()

		data, err := io.ReadAll(src)
		if err != nil {
			return SendError(c, 500, "failed to read file")
		}

		meta, err := h.fileStore.SaveTemp(ctx, file.Filename, data)
		if err != nil {
			return SendError(c, 500, "failed to save file")
		}

		uploaded = append(uploaded, meta)
	}

	return SendSuccess(c, fiber.StatusOK, fiber.Map{
		"files": uploaded,
	}, "files uploaded")
}
