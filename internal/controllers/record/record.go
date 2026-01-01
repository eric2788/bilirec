package record

import (
	"strconv"

	"github.com/eric2788/bilirec/internal/services/recorder"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "record")

type Controller struct {
	service *recorder.Service
}

func NewController(app *fiber.App, service *recorder.Service) *Controller {
	rc := &Controller{service: service}
	record := app.Group("/record")
	record.Post("/:roomID/start", rc.startRecording)
	record.Post("/:roomID/stop", rc.stopRecording)
	record.Get("/:roomID/status", rc.getRecordingStatus)
	record.Get("/:roomID/stats", rc.getRecordingStats)
	record.Get("/list", rc.listRecordings)
	return rc
}

// @Summary Start recording a live stream
// @Description Start recording a Bilibili live stream for the specified room
// @Tags record
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 "Recording started successfully"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 500 {string} string "Internal server error"
// @Router /record/{roomID}/start [post]
func (r *Controller) startRecording(ctx fiber.Ctx) error {
	roomId, err := strconv.ParseInt(ctx.Params("roomID"), 10, 64)
	if err != nil {
		logger.Warnf("cannot parse roomId to int64: %v", err)
		return fiber.ErrBadRequest
	}
	err = r.service.Start(roomId)
	if err != nil {
		logger.Errorf("error starting recording for room %d: %v", roomId, err)
		return fiber.ErrInternalServerError
	}
	return ctx.SendStatus(200)
}

// @Summary Stop recording a live stream
// @Description Stop recording a Bilibili live stream for the specified room
// @Tags record
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {object} map[string]interface{} "Recording stopped"
// @Failure 400 {string} string "Invalid room ID"
// @Router /record/{roomID}/stop [post]
func (r *Controller) stopRecording(ctx fiber.Ctx) error {
	roomId, err := strconv.ParseInt(ctx.Params("roomID"), 10, 64)
	if err != nil {
		logger.Warnf("cannot parse roomId to int64: %v", err)
		return fiber.ErrBadRequest
	}
	stopped := r.service.Stop(roomId)
	return ctx.JSON(fiber.Map{
		"roomID":  roomId,
		"success": stopped,
	})
}

// @Summary Get recording status
// @Description Get the current recording status for a specific room
// @Tags record
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {object} map[string]interface{} "Recording status"
// @Failure 400 {string} string "Invalid room ID"
// @Router /record/{roomID}/status [get]
func (r *Controller) getRecordingStatus(ctx fiber.Ctx) error {
	roomId, err := strconv.ParseInt(ctx.Params("roomID"), 10, 64)
	if err != nil {
		logger.Warnf("cannot parse roomId to int64: %v", err)
		return fiber.ErrBadRequest
	}
	status := r.service.GetStatus(roomId)
	return ctx.JSON(fiber.Map{
		"roomID": roomId,
		"status": status,
	})
}

// @Summary Get recording statistics
// @Description Get recording statistics for a specific room
// @Tags record
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {object} map[string]interface{} "Recording statistics"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 404 {string} string "Recording not found"
// @Router /record/{roomID}/stats [get]
func (r *Controller) getRecordingStats(ctx fiber.Ctx) error {
	roomId, err := strconv.ParseInt(ctx.Params("roomID"), 10, 64)
	if err != nil {
		logger.Warnf("cannot parse roomId to int64: %v", err)
		return fiber.ErrBadRequest
	}
	stats, ok := r.service.GetStats(roomId)
	if !ok {
		return fiber.ErrNotFound
	}
	return ctx.JSON(stats)
}

// @Summary List all recordings
// @Description Get a list of all room IDs that are currently being recorded
// @Tags record
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {array} int64 "List of room IDs"
// @Router /record/list [get]
func (r *Controller) listRecordings(ctx fiber.Ctx) error {
	roomIds := r.service.ListRecording()
	return ctx.JSON(roomIds)
}
