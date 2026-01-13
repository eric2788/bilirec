package file

import (
	"net/url"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/eric2788/bilirec/internal/services/file"
	"github.com/eric2788/bilirec/internal/services/path"
	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "file")

type Controller struct {
	fileSvc     *file.Service
	recorderSvc *recorder.Service
	pathSvc     *path.Service
}

func NewController(
	app *fiber.App,
	fileSvc *file.Service,
	recorderSvc *recorder.Service,
	pathSvc *path.Service,
) *Controller {
	fc := &Controller{
		fileSvc:     fileSvc,
		recorderSvc: recorderSvc,
		pathSvc:     pathSvc,
	}
	files := app.Group("/files")

	files.Get("/browse/*", fc.listFiles)
	files.Get("/download/*", fc.downloadFile)
	files.Get("/tempdownload", fc.presignedDownload)
	files.Post("/presigned/*", fc.createPresignedURL)

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
// @Router /files/browse/{path} [get]
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
// @Success 200 {file} binary "File stream"
// @Failure 400 {string} string "Bad request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/download/{path} [get]
func (c *Controller) downloadFile(ctx fiber.Ctx) error {
	raw := ctx.Params("*", "/")
	path, err := url.PathUnescape(raw)
	if err != nil {
		return fiber.ErrBadRequest
	}
	if c.recorderSvc.IsRecording(path) {
		return fiber.NewError(fiber.StatusBadRequest, "無法下載正在錄製的文件")
	}
	fullPath, err := c.pathSvc.ValidatePath(path)
	if err != nil {
		logger.Warnf("error validating path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	ctx.Attachment(fullPath) // set this because SendFile does not set the filename when using Download: true
	return ctx.SendFile(fullPath, fiber.SendFile{
		ByteRange: true,
	})
}

// @Summary Presigned download
// @Description Download a file using a presigned token (no auth required)
// @Tags files
// @Accept json
// @Produce octet-stream
// @Param presigned query string true "Presigned URL token"
// @Success 200 {file} binary "File stream"
// @Failure 400 {string} string "Bad request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/tempdownload [get]
func (c *Controller) presignedDownload(ctx fiber.Ctx) error {
	token := ctx.Query("presigned", "")
	if token == "" {
		return fiber.ErrBadRequest
	}
	relPath, err := c.pathSvc.ParsePresignedURLToken(token)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	fullPath, err := c.pathSvc.ValidatePath(relPath)
	if err != nil {
		logger.Warnf("error validating path %s: %v", relPath, err)
		return c.parseFiberError(err)
	}
	ctx.Attachment(fullPath) // set this because SendFile does not set the filename when using Download: true
	return ctx.SendFile(fullPath, fiber.SendFile{
		ByteRange: true,
	})
}

// @Summary Create presigned URL
// @Description Create a presigned token for downloading a file. Accepts optional "ttl" query in seconds (default 3600).
// @Tags files
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param path path string true "File path"
// @Param ttl query int false "TTL in seconds"
// @Success 201 {object} PresignedURLResponse "Presigned URL response"
// @Failure 400 {string} string "Bad request"
// @Failure 403 {string} string "Forbidden"
// @Failure 404 {string} string "Not found"
// @Router /files/presigned/{path} [post]
func (c *Controller) createPresignedURL(ctx fiber.Ctx) error {
	raw := ctx.Params("*", "/")
	path, err := url.PathUnescape(raw)
	if err != nil {
		return fiber.ErrBadRequest
	} else if c.recorderSvc.IsRecording(path) {
		return fiber.NewError(fiber.StatusBadRequest, "無法為正在錄製的文件產生臨時下載連結")
	}

	fullPath, err := c.pathSvc.ValidatePath(path)
	if err != nil {
		logger.Warnf("error validating path %s: %v", path, err)
		return c.parseFiberError(err)
	}

	ttlStr := ctx.Query("ttl", "")
	ttlSeconds := int64(3600)
	if ttlStr != "" {
		n, err := strconv.ParseInt(ttlStr, 10, 64)
		if err != nil || n <= 0 {
			return fiber.NewError(fiber.StatusBadRequest, "invalid ttl")
		}
		ttlSeconds = n
	}
	ttl := time.Duration(ttlSeconds) * time.Second

	url, err := c.pathSvc.GeneratePresignedURL(fullPath, ttl)
	if err != nil {
		logger.Warnf("error creating presigned token for path %s: %v", path, err)
		return fiber.ErrInternalServerError
	}

	resp := &PresignedURLResponse{
		URL:       url,
		ExpiresIn: int(ttl.Seconds()),
	}
	return ctx.Status(fiber.StatusCreated).JSON(resp)
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
	case os.IsPermission(err), err == path.ErrAccessDenied:
		return fiber.NewError(fiber.StatusForbidden, "無法存取該文件路徑")
	case err == path.ErrInvalidFilePath:
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
