package file

import (
	"os"

	"github.com/eric2788/bilirec/internal/services/file"
	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "file")

type Controller struct {
	fileSvc *file.Service
}

func NewController(app *fiber.App, fileSvc *file.Service) *Controller {
	fc := &Controller{
		fileSvc: fileSvc,
	}
	files := app.Group("/files")
	files.Get("/*", fc.listFiles)
	files.Post("/*", fc.downloadFile)
	return fc
}

func (c *Controller) listFiles(ctx *fiber.Ctx) error {
	path := ctx.Params("*", "/")
	trees, err := c.fileSvc.ListTree(path)
	if err != nil {
		logger.Warnf("error listing dir at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	return ctx.JSON(trees)
}

func (c *Controller) downloadFile(ctx *fiber.Ctx) error {
	path := ctx.Params("*", "/")
	f, err := c.fileSvc.GetFileStream(path)
	if err != nil {
		logger.Warnf("error getting file stream at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	defer f.Close()
	return ctx.SendStream(f)
}

func (c *Controller) parseFiberError(err error) error {
	switch {
	case os.IsNotExist(err):
		return fiber.ErrNotFound
	case os.IsPermission(err), err == file.ErrAccessDenied:
		return fiber.ErrForbidden
	case err == file.ErrInvalidFilePath:
		return fiber.ErrBadRequest
	case err == file.ErrIsDirectory:
		return fiber.ErrBadRequest
	default:
		return fiber.ErrInternalServerError
	}
}
