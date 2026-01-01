package bilibili

import (
	"strings"

	bili "github.com/CuteReimu/bilibili/v2"
)

func (c *Client) IsStreamLiving(roomID int64) (bool, error) {
	info, err := c.GetLiveRoomInfo(bili.GetLiveRoomInfoParam{
		RoomId: int(roomID),
	})
	if err != nil {
		// very hardcoded detection, hope author can fix that:
		if strings.Contains(err.Error(), "cannot unmarshal array into Go struct field commonResp") {
			return false, ErrRoomNotFound
		}
		return false, err
	}
	return info.LiveStatus == 1, nil
}

func IsErrRoomNotFound(err error) bool {
	return err == ErrRoomNotFound || strings.Contains(err.Error(), "cannot unmarshal array into Go struct field commonResp") // very hardcoded detection, hope author can fix that:
}
