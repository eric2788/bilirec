package room

import (
	"errors"
	"fmt"
	"strconv"

	"go.etcd.io/bbolt"
)

const roomSubscribeBucket = "Room_Subscriptions"

var (
	void = []byte{}

	ErrRoomNotSubscribed     = errors.New("room not subscribed")
	ErrRoomAlreadySubscribed = errors.New("room already subscribed")
)

func (r *Service) Subscribe(roomID int) error {
	// room existence check
	if _, err := r.GetLiveRoomInfo(roomID); err != nil {
		return err
	}
	key := fmt.Append(nil, roomID)
	return r.bucket.Update(func(bucket *bbolt.Bucket) error {
		exists := bucket.Get(key)
		if exists != nil {
			return ErrRoomAlreadySubscribed
		}
		return bucket.Put(key, void)
	})
}

func (r *Service) Unsubscribe(roomID int) error {
	key := fmt.Append(nil, roomID)
	return r.bucket.Update(func(bucket *bbolt.Bucket) error {
		exists := bucket.Get(key)
		if exists == nil {
			return ErrRoomNotSubscribed
		}
		return bucket.Delete(key)
	})
}

func (r *Service) IsSubscribed(roomID int) (bool, error) {
	key := fmt.Append(nil, roomID)
	return r.bucket.Exists(key)
}

func (r *Service) ListSubscribedRooms() ([]int, error) {
	var roomIDs []int
	err := r.bucket.ForEach(func(k, v []byte) error {
		roomID, err := strconv.Atoi(string(k))
		if err != nil {
			logger.Warnf("error scaning item: %s: %v, ignored.", string(k), err)
			return nil
		}
		roomIDs = append(roomIDs, roomID)
		return nil
	})
	return roomIDs, err
}
