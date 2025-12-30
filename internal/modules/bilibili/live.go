package bilibili

import bili "github.com/CuteReimu/bilibili/v2"

func (c *Client) IsStreamLiving(roomID int64) (bool, error) {
	info, err := c.GetLiveRoomInfo(bili.GetLiveRoomInfoParam{
		RoomId: int(roomID),
	})
	if err != nil {
		return false, err
	}
	return info.LiveStatus == 1, nil
}
