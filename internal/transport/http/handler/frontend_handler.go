package handler

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"

	"github.com/gofiber/fiber/v2"
)

type FrontendHandler struct {
	frontendFS embed.FS
}

func NewFrontendHandler(
	frontendFS embed.FS,
) *FrontendHandler {
	return &FrontendHandler{
		frontendFS: frontendFS,
	}
}

func (h *FrontendHandler) Init(app *fiber.App) {
	sub, err := fs.Sub(h.frontendFS, "frontend/dist")
	if err != nil {
		panic(err)
	}

	app.Use(func(c *fiber.Ctx) error {
		reqPath := c.Path()

		if reqPath == "/" {
			reqPath = "/index.html"
		}

		cleanPath := path.Clean(reqPath)
		file := strings.TrimPrefix(cleanPath, "/")

		data, err := fs.ReadFile(sub, file)
		if err != nil {
			fmt.Println(err)
			if strings.HasPrefix(file, "assets/") ||
				strings.Contains(file, ".") {
				return c.SendStatus(404)
			}

			data, err = fs.ReadFile(sub, "index.html")
			if err != nil {
				return c.SendStatus(404)
			}
			c.Type("html")
			return c.Send(data)
		}

		c.Type(filepath.Ext(file))
		return c.Send(data)
	})
}
