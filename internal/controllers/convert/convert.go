package convert

import (
	"github.com/eric2788/bilirec/internal/services/convert"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "convert")

type Controller struct {
	convertSvc *convert.Service
}

func NewController(app *fiber.App, convertSvc *convert.Service) *Controller {
	cc := &Controller{
		convertSvc: convertSvc,
	}

	converts := app.Group("/convert")
	converts.Get("/tasks", cc.listConvertTasks)
	converts.Delete("/tasks/:task_id", cc.cancelTask)
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
