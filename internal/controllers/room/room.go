package room

import (
	"strconv"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "room")

type Controller struct {
	bilic *bilibili.Client
}

func NewController(app *fiber.App, bilic *bilibili.Client) *Controller {
	rc := &Controller{
		bilic: bilic,
	}
	room := app.Group("/room")
	room.Get("/:roomID/info", rc.getRoomInfo)
	room.Get("/:roomID/live", rc.isStreamLiving)
	return rc
}

// @Summary Get room information
// @Description Get detailed information about a Bilibili live room
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {object} bilibili.LiveRoomInfoDetail "Room information"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 500 {string} string "Internal server error"
// @Router /room/{roomID}/info [get]
func (r *Controller) getRoomInfo(ctx fiber.Ctx) error {
	roomId, err := strconv.Atoi(ctx.Params("roomID"))
	if err != nil {
		logger.Warnf("cannot parse roomId to int: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "無效的房間 ID")
	}
	res, err := r.bilic.GetLiveRoomInfo(roomId)

	if err != nil {
		logger.Errorf("error getting room info for room %d: %v", roomId, err)
		return utils.Ternary(
			bilibili.IsErrRoomNotFound(err),
			fiber.NewError(fiber.StatusNotFound, "房間不存在"),
			fiber.ErrInternalServerError,
		)
	}

	return ctx.JSON(res)
}

// @Summary Check if stream is live
// @Description Check if a Bilibili live stream is currently live
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {object} LiveInfo "Live status"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 500 {string} string "Internal server error"
// @Router /room/{roomID}/live [get]
func (r *Controller) isStreamLiving(ctx fiber.Ctx) error {
	roomId, err := strconv.Atoi(ctx.Params("roomID"))
	if err != nil {
		logger.Warnf("cannot parse roomId to int: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "無效的房間 ID")
	}
	isLive, err := r.bilic.IsStreamLiving(roomId)
	if err != nil {
		logger.Errorf("error checking stream living status for room %d: %v", roomId, err)
		return utils.Ternary(
			bilibili.IsErrRoomNotFound(err),
			fiber.NewError(fiber.StatusNotFound, "房間不存在"),
			fiber.ErrInternalServerError,
		)
	}
	return ctx.JSON(LiveInfo{
		RoomId: roomId,
		IsLive: isLive,
	})
}
