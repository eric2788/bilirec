package convert

import (
	"net/url"
	"os"

	"github.com/eric2788/bilirec/internal/services/convert"
	"github.com/eric2788/bilirec/internal/services/path"
	"github.com/eric2788/bilirec/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "convert")

type Controller struct {
	convertSvc *convert.Service
	pathSvc    *path.Service
}

func NewController(app *fiber.App, convertSvc *convert.Service, pathSvc *path.Service) *Controller {
	cc := &Controller{
		convertSvc: convertSvc,
		pathSvc:    pathSvc,
	}

	converts := app.Group("/convert")
	converts.Get("/tasks", cc.listConvertTasks)
	converts.Delete("/tasks/:task_id", cc.cancelTask)
	converts.Post("/tasks/*", cc.enqueueTask)
	return cc
}

// @Summary List in-progress convert tasks
// @Description List currently in-progress convert tasks
// @Tags convert
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} convert.TaskQueue "List of convert tasks"
// @Failure 500 {string} string "Internal server error"
// @Router /convert/tasks [get]
func (c *Controller) listConvertTasks(ctx fiber.Ctx) error {
	tasks, err := c.convertSvc.ListInProgress()
	if err != nil {
		logger.Errorf("error listing convert tasks: %v", err)
		return fiber.ErrInternalServerError
	}

	// make them relative
	for i := range tasks {

		if rel, err := c.pathSvc.GetRelativePath(tasks[i].InputPath); err == nil {
			tasks[i].InputPath = rel
		} else {
			logger.Warnf("error getting relative path for %s: %v", tasks[i].InputPath, err)
		}

		if rel, err := c.pathSvc.GetRelativePath(tasks[i].OutputPath); err == nil {
			tasks[i].OutputPath = rel
		} else {
			logger.Warnf("error getting relative path for %s: %v", tasks[i].OutputPath, err)
		}

	}
	return ctx.JSON(tasks)
}

// @Summary Cancel convert task
// @Description Cancel an in-progress convert task by id
// @Tags convert
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param task_id path string true "Task ID"
// @Success 204 {string} string "No Content"
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 500 {string} string "Internal server error"
// @Router /convert/tasks/{task_id} [delete]
func (c *Controller) cancelTask(ctx fiber.Ctx) error {
	taskID := ctx.Params("task_id", "")
	if taskID == "" {
		return fiber.ErrBadRequest
	}
	if err := c.convertSvc.Cancel(taskID); err != nil {
		if err == convert.ErrTaskNotFound {
			return fiber.ErrNotFound
		} else {
			logger.Errorf("error cancelling convert task %s: %v", taskID, err)
			return fiber.ErrInternalServerError
		}
	}
	return ctx.SendStatus(fiber.StatusNoContent)
}

// @Summary Enqueue convert task
// @Description Enqueue a new convert task for the given video file path
// @Tags convert
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param path path string true "Video file path"
// @Param delete query bool false "Whether to delete the original file after conversion" default(false)
// @Success 200 {object} convert.TaskQueue "Enqueued convert task"
// @Failure 400 {string} string "Bad Request"
// @Failure 404 {string} string "Not Found"
// @Failure 409 {string} string "Conflict: File already in convert queue"
// @Failure 403 {string} string "Forbidden"
// @Failure 500 {string} string "Internal server error"
// @Router /convert/tasks/{path} [post]
func (c *Controller) enqueueTask(ctx fiber.Ctx) error {
	raw := ctx.Params("*", "/")
	path, err := url.PathUnescape(raw)
	if err != nil {
		return fiber.ErrBadRequest
	}
	delete := ctx.Query("delete", "false") == "true"
	fullPath, err := c.pathSvc.ValidatePath(path)
	if err != nil {
		logger.Warnf("error validating file path %s: %v", path, err)
		return c.parseFiberError(err)
	}
	if inQueue, err := c.convertSvc.IsInQueue(fullPath); err != nil {
		logger.Errorf("error checking if path %s is in convert queue: %v", path, err)
		return c.parseFiberError(err)
	} else if inQueue {
		return fiber.NewError(fiber.StatusConflict, "該文件已在轉檔佇列中")
	}
	// currnetly only support converting to mp4
	if q, err := c.convertSvc.Enqueue(fullPath, "mp4", delete); err != nil {
		logger.Errorf("error enqueueing convert task for path %s: %v", path, err)
		return utils.Ternary(
			os.IsNotExist(err),
			fiber.ErrNotFound,
			fiber.ErrInternalServerError)
	} else {
		return ctx.JSON(q)
	}
}

func (c *Controller) parseFiberError(err error) error {
	switch {
	case os.IsNotExist(err):
		return fiber.NewError(fiber.StatusNotFound, "找不到所屬文件夾或檔案")
	case os.IsPermission(err), err == path.ErrAccessDenied:
		return fiber.NewError(fiber.StatusForbidden, "無法存取該文件路徑")
	case err == path.ErrInvalidFilePath:
		return fiber.NewError(fiber.StatusBadRequest, "無效文件路徑")
	default:
		return fiber.ErrInternalServerError
	}
}
