package room

import (
	"strconv"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/eric2788/bilirec/internal/services/room"
	"github.com/eric2788/bilirec/utils"
	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
)

var logger = logrus.WithField("controller", "room")

type Controller struct {
	roomSvc *room.Service
}

func NewController(app *fiber.App, roomSvc *room.Service) *Controller {
	rc := &Controller{
		roomSvc: roomSvc,
	}
	room := app.Group("/room")
	room.Get("/:roomID/info", rc.getRoomInfo)
	room.Get("/infos", rc.getRoomInfos)
	room.Get("/:roomID/live", rc.isStreamLiving)

	room.Get("/subscribe", rc.listSubscribeRooms)
	room.Get("/subscribe/:roomID", rc.isSubscribeRoom)
	room.Post("/:roomID", rc.subscribeRoom)
	room.Delete("/:roomID", rc.unsubscribeRoom)
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
	res, err := r.roomSvc.GetLiveRoomInfo(roomId)

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
	res, err := r.roomSvc.GetMultipleRoomInfos(roomIds...)
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
	isLive, err := r.roomSvc.IsRoomLive(roomId)
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

// @Summary Subscribe to room
// @Description Subscribe to a Bilibili live room for updates
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {string} string "Subscription successful"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 409 {string} string "Already subscribed"
// @Failure 500 {string} string "Internal server error"
// @Router /room/{roomID} [post]
func (r *Controller) subscribeRoom(ctx fiber.Ctx) error {
	roomId, err := strconv.Atoi(ctx.Params("roomID"))
	if err != nil {
		logger.Warnf("cannot parse roomId to int: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "無效的房間 ID")
	}
	err = r.roomSvc.Subscribe(roomId)
	if err != nil {
		logger.Errorf("error subscribing to room %d: %v", roomId, err)
		return utils.Ternary(
			room.ErrRoomAlreadySubscribed == err,
			fiber.NewError(fiber.StatusConflict, "已訂閱此房間"),
			fiber.ErrInternalServerError,
		)
	}
	return ctx.SendStatus(fiber.StatusOK)
}

// @Summary Unsubscribe from room
// @Description Unsubscribe from a Bilibili live room
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {string} string "Unsubscription successful"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 404 {string} string "Not subscribed"
// @Failure 500 {string} string "Internal server error"
// @Router /room/{roomID} [delete]
func (r *Controller) unsubscribeRoom(ctx fiber.Ctx) error {
	roomId, err := strconv.Atoi(ctx.Params("roomID"))
	if err != nil {
		logger.Warnf("cannot parse roomId to int: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "無效的房間 ID")
	}
	err = r.roomSvc.Unsubscribe(roomId)
	if err != nil {
		logger.Errorf("error unsubscribing from room %d: %v", roomId, err)
		return utils.Ternary(
			room.ErrRoomNotSubscribed == err,
			fiber.NewError(fiber.StatusNotFound, "未訂閱此房間"),
			fiber.ErrInternalServerError,
		)
	}
	return ctx.SendStatus(fiber.StatusOK)
}

// @Summary Check if room is subscribed
// @Description Check if a Bilibili live room is subscribed
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param roomID path int true "Room ID"
// @Success 200 {object} SubscribeStatus "Subscription status"
// @Failure 400 {string} string "Invalid room ID"
// @Failure 500 {string} string "Internal server error"
// @Router /room/subscribe/{roomID} [get]
func (r *Controller) isSubscribeRoom(ctx fiber.Ctx) error {
	roomId, err := strconv.Atoi(ctx.Params("roomID"))
	if err != nil {
		logger.Warnf("cannot parse roomId to int: %v", err)
		return fiber.NewError(fiber.StatusBadRequest, "無效的房間 ID")
	}
	isSubscribed, err := r.roomSvc.IsSubscribed(roomId)
	if err != nil {
		logger.Errorf("error checking subscription status for room %d: %v", roomId, err)
		return fiber.ErrInternalServerError
	}
	return ctx.JSON(SubscribeStatus{
		RoomId:       roomId,
		IsSubscribed: isSubscribed,
	})
}

// @Summary List subscribed rooms
// @Description List all subscribed Bilibili live rooms
// @Tags room
// @Security BearerAuth
// @Accept json
// @Produce json
// @Success 200 {object} SubscribeList "List of subscribed Room IDs"
// @Failure 500 {string} string "Internal server error"
// @Router /room/subscribe [get]
func (r *Controller) listSubscribeRooms(ctx fiber.Ctx) error {
	roomIds, err := r.roomSvc.ListSubscribedRooms()
	if err != nil {
		logger.Errorf("error listing subscribed rooms: %v", err)
		return fiber.ErrInternalServerError
	}
	return ctx.JSON(SubscribeList{
		RoomIds: roomIds,
	})
}
