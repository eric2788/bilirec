package file

import (
	"os"

	"github.com/eric2788/bilirec/internal/services/file"
	"github.com/gofiber/fiber/v3"
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

	files.Delete("/*", fc.deleteDir)
	files.Delete("/batch", fc.deleteFiles)
	
	return fc
}

func (c *Controller) listFiles(ctx fiber.Ctx) error {
	path := ctx.Params("*", "/")
	trees, err := c.fileSvc.ListTree(path)
	if err != nil {
		logger.Warnf("error listing dir at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	return ctx.JSON(trees)
}

func (c *Controller) downloadFile(ctx fiber.Ctx) error {
	path := ctx.Params("*", "/")
	format := ctx.Query("format", "flv")
	f, err := c.fileSvc.GetFileStream(path, format)
	if err != nil {
		logger.Warnf("error getting file stream at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	defer f.Close()
	return ctx.SendStream(f)
}

func (c *Controller) deleteFiles(ctx fiber.Ctx) error {
	var paths []string
	if err := ctx.Bind().Body(&paths); err != nil {
		return fiber.ErrBadRequest
	}
	if err := c.fileSvc.DeleteFiles(paths...); err != nil {
		logger.Warnf("error deleting files: %v", err)
		return c.parseFiberError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *Controller) deleteDir(ctx fiber.Ctx) error {
	path := ctx.Params("*", "/")
	if err := c.fileSvc.DeleteDirectory(path); err != nil {
		logger.Warnf("error deleting directory at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
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
