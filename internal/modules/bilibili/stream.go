package bilibili

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-resty/resty/v2"
)

type (
	StreamResponseV1 struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		TTL     int        `json:"ttl"`
		Data    StreamData `json:"data"`
	}

	StreamData struct {
		CurrentQuality     int           `json:"current_quality"`
		AcceptQuality      []string      `json:"accept_quality"`
		CurrentQn          int           `json:"current_qn"`
		QualityDescription []QualityDesc `json:"quality_description"`
		Durl               []StreamURL   `json:"durl"`
	}

	QualityDesc struct {
		Qn   int    `json:"qn"`
		Desc string `json:"desc"`
	}

	StreamURL struct {
		URL        string `json:"url"`
		Length     int    `json:"length"`
		Order      int    `json:"order"`
		StreamType int    `json:"stream_type"`
		P2PType    int    `json:"p2p_type"`
	}
)

type (
	StreamResponseV2 struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		TTL     int    `json:"ttl"`
		Data    struct {
			PlayurlInfo *PlayurlInfo `json:"playurl_info"`
		} `json:"data"`
	}

	RoomPlayInfo struct {
		LiveStatus  int          `json:"live_status"`
		Encrypted   bool         `json:"encrypted"`
		PlayurlInfo *PlayurlInfo `json:"playurl_info"`
	}

	PlayurlInfo struct {
		Playurl *Playurl `json:"playurl"`
	}

	Playurl struct {
		Streams []StreamItem `json:"stream"`
	}

	StreamItem struct {
		ProtocolName string       `json:"protocol_name"`
		Formats      []FormatItem `json:"format"`
	}

	FormatItem struct {
		FormatName string      `json:"format_name"`
		Codecs     []CodecItem `json:"codec"`
	}

	CodecItem struct {
		CodecName string        `json:"codec_name"`
		BaseUrl   string        `json:"base_url"`
		CurrentQn int           `json:"current_qn"`
		AcceptQn  []int         `json:"accept_qn"`
		UrlInfos  []UrlInfoItem `json:"url_info"`
	}

	UrlInfoItem struct {
		Host  string `json:"host"`
		Extra string `json:"extra"`
	}
)

const v1StreamAPI = "https://api.live.bilibili.com/room/v1/Room/playUrl"
const v2StreamAPI = "https://api.live.bilibili.com/xlive/web-room/v2/index/getRoomPlayInfo"

func (c *Client) GetStreamURLs(roomID int64) ([]string, error) {
	client := c.liveClient.R()
	client.SetQueryParams(map[string]string{
		"cid":      fmt.Sprint(roomID),
		"qn":       "10000",
		"platform": "web",
	})

	resp, err := client.Get(v1StreamAPI)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode())
	}

	var sr StreamResponseV1
	if err := json.Unmarshal(resp.Body(), &sr); err != nil {
		return nil, err
	} else if sr.Code != 0 {
		return nil, fmt.Errorf("error getting stream url: %s (code %d)", sr.Message, sr.Code)
	}

	urls := make([]string, 0, len(sr.Data.Durl))
	for _, durl := range sr.Data.Durl {
		urls = append(urls, durl.URL)
	}
	return urls, nil
}

func (c *Client) GetStreamURLsV2(roomID int64) ([]string, error) {
	client := c.liveClient.R()
	client.SetQueryParams(map[string]string{
		"room_id":      fmt.Sprint(roomID),
		"qn":           "10000",
		"no_playurl":   "0",
		"mask":         "1",
		"platform":     "web",
		"protocol":     "0,1",
		"format":       "0,1,2",
		"codec":        "0,1,2",
		"dolby":        "5",
		"panorama":     "1",
		"hdr_type":     "0,1",
		"web_location": "444.8",
	})
	newQueryParam, err := c.wbi.SignQuery(client.QueryParam, time.Now())
	if err != nil {
		return nil, fmt.Errorf("cannot sign wbi: %v", err)
	}
	client.QueryParam = newQueryParam

	resp, err := client.Get(v2StreamAPI)
	if err != nil {
		return nil, err
	} else if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("status code: %d", resp.StatusCode())
	}

	var sr StreamResponseV2
	if err := json.Unmarshal(resp.Body(), &sr); err != nil {
		return nil, err
	} else if sr.Code != 0 {
		return nil, fmt.Errorf("error getting stream url: %s (code %d)", sr.Message, sr.Code)
	}

	urls := make([]string, 0)

	if sr.Data.PlayurlInfo == nil || sr.Data.PlayurlInfo.Playurl == nil {
		return urls, nil
	}

	for _, stream := range sr.Data.PlayurlInfo.Playurl.Streams {
		for _, format := range stream.Formats {
			for _, codec := range format.Codecs {
				for _, urlInfo := range codec.UrlInfos {
					fullURL := urlInfo.Host + codec.BaseUrl + urlInfo.Extra
					urls = append(urls, fullURL)
				}
			}
		}
	}

	return urls, nil
}

func (c *Client) FetchLiveStreamUrl(url string) (*resty.Response, error) {
	return c.DoLiveStream(func(req *resty.Request) (*resty.Response, error) {
		return req.Get(url)
	})
}
