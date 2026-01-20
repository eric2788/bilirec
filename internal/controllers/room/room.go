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
	room.Get("/infos", rc.getRoomInfos)
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

// @Summary Get multiple room informations
// @Description Get detailed information about multiple Bilibili live rooms
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomIDs query string true "Comma-separated list of Room IDs"
// @Success 200 {object} map[string]bilibili.LiveRoomInfoDetail "Map of Room ID to Room information"
// @Failure 400 {string} string "Invalid room IDs"
// @Failure 500 {string} string "Internal server error"
// @Router /room/infos [get]
func (r *Controller) getRoomInfos(ctx fiber.Ctx) error {
	roomIdsStr := ctx.Query("roomIDs", "")
	if roomIdsStr == "" {
		return fiber.NewError(fiber.StatusBadRequest, "缺少 roomIDs 查詢參數")
	}
	roomIdStrs := utils.SplitAndTrim(roomIdsStr, ",")
	roomIds := make([]int, 0, len(roomIdStrs))
	for _, idStr := range roomIdStrs {
		id, err := strconv.Atoi(idStr)
		if err != nil {
			logger.Warnf("cannot parse roomId to int: %v", err)
			return fiber.NewError(fiber.StatusBadRequest, "無效的房間 ID: "+idStr)
		}
		roomIds = append(roomIds, id)
	}
	res, err := r.bilic.GetLiveRoomInfos(roomIds...)
	if err != nil {
		logger.Errorf("error getting room infos for rooms %v: %v", roomIds, err)
		return utils.Ternary(
			bilibili.IsErrRoomNotFound(err),
			fiber.NewError(fiber.StatusNotFound, "部分或全部房間不存在"),
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
