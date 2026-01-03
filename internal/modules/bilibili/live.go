package bilibili

import (
	bili "github.com/CuteReimu/bilibili/v2"
	"github.com/pkg/errors"
)

func (c *Client) IsStreamLiving(roomID int) (bool, error) {
	info, err := c.GetLiveRoomInfo(roomID)
	if err != nil {
		return false, err
	}
	return info.LiveStatus == 1, nil
}

func (c *Client) GetLiveRoomInfo(roomId int) (*bili.LiveRoomInfo, error) {
	info, err := c.Client.GetLiveRoomInfo(bili.GetLiveRoomInfoParam{
		RoomId: roomId,
	})
	if err != nil {
		if IsErrRoomNotFound(err) {
			return nil, ErrRoomNotFound
		}
		return nil, err
	}
	return info, nil
}

func IsErrRoomNotFound(err error) bool {
	if err == ErrRoomNotFound {
		return true
	}
	cause := errors.Cause(err)
	if biliErr, ok := cause.(bili.Error); ok && biliErr.Code == 1 {
		return true
	}
	return false
}
