package room

import bili "github.com/CuteReimu/bilibili/v2"

type (
	LiveRoomInfo struct {
		Uid            int    `json:"uid"`              // 主播mid
		RoomId         int    `json:"room_id"`          // 直播间长号
		ShortId        int    `json:"short_id"`         // 直播间短号。为0是无短号
		Attention      int    `json:"attention"`        // 关注数量
		Online         int    `json:"online"`           // 观看人数
		IsPortrait     bool   `json:"is_portrait"`      // 是否竖屏
		Description    string `json:"description"`      // 描述
		LiveStatus     int    `json:"live_status"`      // 直播状态。0：未开播。1：直播中。2：轮播中
		AreaId         int    `json:"area_id"`          // 分区id
		ParentAreaId   int    `json:"parent_area_id"`   // 父分区id
		ParentAreaName string `json:"parent_area_name"` // 父分区名称
		OldAreaId      int    `json:"old_area_id"`      // 旧版分区id
		Background     string `json:"background"`       // 背景图片链接
		Title          string `json:"title"`            // 标题
		UserCover      string `json:"user_cover"`       // 封面
		LiveTime       string `json:"live_time"`        // 直播开始时间。YYYY-MM-DD HH:mm:ss
		Tags           string `json:"tags"`             // 标签。','分隔
		AreaName       string `json:"area_name"`        // 分区名称
	}

	LiveInfo struct {
		RoomId int  `json:"room_id"`
		IsLive bool `json:"is_live"`
	}
)

func newLiveRoomInfo(info *bili.LiveRoomInfo) *LiveRoomInfo {
	return &LiveRoomInfo{
		Uid:            info.Uid,
		RoomId:         info.RoomId,
		ShortId:        info.ShortId,
		Attention:      info.Attention,
		Online:         info.Online,
		IsPortrait:     info.IsPortrait,
		Description:    info.Description,
		LiveStatus:     info.LiveStatus,
		AreaId:         info.AreaId,
		ParentAreaId:   info.ParentAreaId,
		ParentAreaName: info.ParentAreaName,
		OldAreaId:      info.OldAreaId,
		Background:     info.Background,
		Title:          info.Title,
		UserCover:      info.UserCover,
		LiveTime:       info.LiveTime,
		Tags:           info.Tags,
		AreaName:       info.AreaName,
	}
}
