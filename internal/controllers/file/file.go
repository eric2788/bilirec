package file

import (
	"net/url"
	"os"
	"slices"

	"github.com/eric2788/bilirec/internal/services/file"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "file")

type Controller struct {
	fileSvc     *file.Service
	recorderSvc *recorder.Service
}

func NewController(app *fiber.App, fileSvc *file.Service, recorderSvc *recorder.Service) *Controller {
	fc := &Controller{
		fileSvc:     fileSvc,
		recorderSvc: recorderSvc,
	}
	files := app.Group("/files")

	files.Get("/*", fc.listFiles)
	files.Post("/*", fc.downloadFile)

	files.Delete("/batch", fc.deleteFiles)
	files.Delete("/*", fc.deleteDir)

	return fc
}

// @Summary List files and directories
// @Description List files and directories under a given path
// @Tags files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param path path string false "Relative path"
// @Success 200 {array} file.Tree "List of files and directories"
// @Failure 400 {string} string "Invalid path"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/{path} [get]
func (c *Controller) listFiles(ctx fiber.Ctx) error {
	raw := ctx.Params("*", "/")
	path, err := url.PathUnescape(raw)
	if err != nil {
		return fiber.ErrBadRequest
	}
	trees, err := c.fileSvc.ListTree(path)
	if err != nil {
		logger.Warnf("error listing dir at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	return ctx.JSON(c.withRecordingStatus(trees))
}

// @Summary Download a file
// @Description Download a file or convert it to the requested format
// @Tags files
// @Security BearerAuth
// @Accept json
// @Produce octet-stream
// @Param path path string true "File path"
// @Param format query string false "Output format (e.g., flv)"
// @Success 200 {file} binary "File stream"
// @Failure 400 {string} string "Bad request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/{path} [get]
func (c *Controller) downloadFile(ctx fiber.Ctx) error {
	raw := ctx.Params("*", "/")
	path, err := url.PathUnescape(raw)
	if err != nil {
		return fiber.ErrBadRequest
	}
	if c.recorderSvc.IsRecording(path) {
		return fiber.NewError(fiber.StatusBadRequest, "無法下載正在錄製的文件")
	}
	format := ctx.Query("format", "flv")
	f, err := c.fileSvc.GetFileStream(path, format)
	if err != nil {
		logger.Warnf("error getting file stream at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	return ctx.SendStream(f)
}

// @Summary Delete multiple files
// @Description Delete multiple files by their relative paths
// @Tags files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param paths body []string true "List of relative file paths to delete"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/batch [delete]
func (c *Controller) deleteFiles(ctx fiber.Ctx) error {
	var paths []string
	if err := ctx.Bind().Body(&paths); err != nil {
		return fiber.ErrBadRequest
	} else if slices.ContainsFunc(paths, c.recorderSvc.IsRecording) {
		return fiber.NewError(fiber.StatusBadRequest, "要刪除的文件中包含正在錄製的文件")
	} else if err := c.fileSvc.DeleteFiles(paths...); err != nil {
		logger.Warnf("error deleting files: %v", err)
		return c.parseFiberError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// @Summary Delete a directory
// @Description Delete a directory and all its contents
// @Tags files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param path path string true "Directory path"
// @Success 204 "No Content"
// @Failure 400 {string} string "Bad request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/{path} [delete]
func (c *Controller) deleteDir(ctx fiber.Ctx) error {
	raw := ctx.Params("*", "/")
	path, err := url.PathUnescape(raw)
	if err != nil {
		return fiber.ErrBadRequest
	} else if c.recorderSvc.IsRecordingUnder(path) {
		return fiber.NewError(fiber.StatusBadRequest, "無法刪除包含正在錄製文件的文件夾")
	} else if err := c.fileSvc.DeleteDirectory(path); err != nil {
		logger.Warnf("error deleting directory at path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

func (c *Controller) parseFiberError(err error) error {
	switch {
	case os.IsNotExist(err):
		return fiber.NewError(fiber.StatusNotFound, "找不到所屬文件夾或檔案")
	case os.IsPermission(err), err == file.ErrAccessDenied:
		return fiber.NewError(fiber.StatusForbidden, "無法存取該文件路徑")
	case err == file.ErrInvalidFilePath:
		return fiber.NewError(fiber.StatusBadRequest, "無效文件路徑")
	case err == file.ErrIsDirectory:
		return fiber.NewError(fiber.StatusBadRequest, "此路徑為文件夾")
	default:
		return fiber.ErrInternalServerError
	}
}

func (c *Controller) withRecordingStatus(tree []file.Tree) []file.Tree {
	out := make([]file.Tree, len(tree))
	copy(out, tree)
	for i := range out {
		if !out[i].IsDir {
			out[i].IsRecording = c.recorderSvc.IsRecording(out[i].Path)
		}
	}
	return out
}
