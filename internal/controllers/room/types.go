package room

type LiveInfo struct {
	RoomId int  `json:"room_id"`
	IsLive bool `json:"is_live"`
}

type SubscribeStatus struct {
	RoomId       int  `json:"room_id"`
	IsSubscribed bool `json:"is_subscribed"`
}

type SubscribeList struct {
	RoomIds []int `json:"room_ids"`
}
