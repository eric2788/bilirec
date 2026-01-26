package room

import (
	"fmt"

	"github.com/eric2788/bilirec/internal/modules/bilibili"
	"github.com/jellydator/ttlcache/v3"
)

// add some caching

func (r *Service) GetLiveRoomInfo(roomID int) (*bilibili.LiveRoomInfoDetail, error) {
	data := r.cache.Get(fmt.Sprint(roomID))
	if data != nil {
		return data.Value(), nil
	}
	info, err := r.bilic.GetLiveRoomInfo(roomID)
	if err != nil {
		return nil, err
	}
	r.cache.Set(fmt.Sprint(roomID), info, ttlcache.DefaultTTL)
	return info, nil
}

func (r *Service) IsRoomLive(roomID int) (bool, error) {
	info, err := r.GetLiveRoomInfo(roomID)
	if err != nil {
		return false, err
	}
	return info.LiveStatus == 1, nil
}

func (r *Service) GetMultipleRoomInfos(roomIDs ...int) (map[string]*bilibili.LiveRoomInfoDetail, error) {
	infos := make(map[string]*bilibili.LiveRoomInfoDetail)
	missedIDs := make([]int, 0, len(roomIDs))

	// Check cache first
	for _, id := range roomIDs {
		idStr := fmt.Sprint(id)
		if data := r.cache.Get(idStr); data != nil {
			infos[idStr] = data.Value()
		} else {
			missedIDs = append(missedIDs, id)
		}
	}

	// Fetch missing ones
	if len(missedIDs) > 0 {
		fetchedInfos, err := r.bilic.GetLiveRoomInfos(missedIDs...)
		if err != nil {
			return nil, err
		}

		// Cache and add to result
		for id, info := range fetchedInfos {
			r.cache.Set(id, info, ttlcache.DefaultTTL)
			infos[id] = info
		}
	}

	return infos, nil
}
