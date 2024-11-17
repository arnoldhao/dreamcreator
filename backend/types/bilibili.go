package types

// {"code":0,"message":"0","ttl":1,"data":{"token":"aaa"}}
// {"code":-101,"message":"账号未登录","ttl":1}
type BilibiliTokenData struct {
	Token string `json:"token"`
}

type BilibiliToken struct {
	Code    int               `json:"code"`
	Message string            `json:"message"`
	Data    BilibiliTokenData `json:"data"`
}

type BilibiliInteraction struct {
	Interaction bool `json:"interaction"`
}

type BilibiliEpVideoInfo struct {
	Aid                           int                 `json:"aid"`
	Bvid                          string              `json:"bvid"`
	Cid                           int                 `json:"cid"`
	DeliveryBusinessFragmentVideo bool                `json:"delivery_business_fragment_video"`
	DeliveryFragmentVideo         bool                `json:"delivery_fragment_video"`
	EpID                          int                 `json:"ep_id"`
	EpStatus                      int                 `json:"ep_status"`
	Interaction                   BilibiliInteraction `json:"interaction"`
	LongTitle                     string              `json:"long_title"`
	Title                         string              `json:"title"`
}

type BilibiliBangumiData struct {
	EpInfo BilibiliEpVideoInfo   `json:"epInfo"`
	EpList []BilibiliEpVideoInfo `json:"epList"`
}

type BilibiliVideoPagesData struct {
	Cid  int    `json:"cid"`
	Part string `json:"part"`
	Page int    `json:"page"`
}

type BilibiliMultiPageVideoData struct {
	Title string                   `json:"title"`
	Pages []BilibiliVideoPagesData `json:"pages"`
}

type BilibiliEpisode struct {
	Aid   int    `json:"aid"`
	Cid   int    `json:"cid"`
	Title string `json:"title"`
	BVid  string `json:"bvid"`
}

type BilibiliMultiEpisodeData struct {
	Seasionid int               `json:"season_id"`
	Episodes  []BilibiliEpisode `json:"episodes"`
}

type BilibiliMultiPage struct {
	Aid       int                        `json:"aid"`
	BVid      string                     `json:"bvid"`
	Sections  []BilibiliMultiEpisodeData `json:"sections"`
	VideoData BilibiliMultiPageVideoData `json:"videoData"`
}

type BilibiliDashStream struct {
	ID        int    `json:"id"`
	BaseURL   string `json:"baseUrl"`
	Bandwidth int    `json:"bandwidth"`
	MimeType  string `json:"mimeType"`
	Codecid   int    `json:"codecid"`
	Codecs    string `json:"codecs"`
}

type BilibiliDashStreams struct {
	Video []BilibiliDashStream `json:"video"`
	Audio []BilibiliDashStream `json:"audio"`
}

type BilibiliDashInfo struct {
	CurQuality  int                 `json:"quality"`
	Description []string            `json:"accept_description"`
	Quality     []int               `json:"accept_quality"`
	Streams     BilibiliDashStreams `json:"dash"`
	DURLFormat  string              `json:"format"`
	DURLs       []BilibiliDURL      `json:"durl"`
}

type BilibiliDURL struct {
	URL  string `json:"url"`
	Size int64  `json:"size"`
}

type BilibiliDash struct {
	Code    int              `json:"code"`
	Message string           `json:"message"`
	Data    BilibiliDashInfo `json:"data"`
	Result  BilibiliDashInfo `json:"result"`
}

var BilibiliQualityString = map[int]string{
	127: "超高清 8K",
	120: "超清 4K",
	116: "高清 1080P60",
	74:  "高清 720P60",
	112: "高清 1080P+",
	80:  "高清 1080P",
	64:  "高清 720P",
	48:  "高清 720P",
	32:  "清晰 480P",
	16:  "流畅 360P",
	15:  "流畅 360P",
}

type BilibiliSubtitleData struct {
	From     float32 `json:"from"`
	To       float32 `json:"to"`
	Location int     `json:"location"`
	Content  string  `json:"content"`
}

type BilibiliSubtitleFormat struct {
	FontSize        float32                `json:"font_size"`
	FontColor       string                 `json:"font_color"`
	BackgroundAlpha float32                `json:"background_alpha"`
	BackgroundColor string                 `json:"background_color"`
	Stroke          string                 `json:"Stroke"`
	Body            []BilibiliSubtitleData `json:"body"`
}

type BilibiliSubtitleProperty struct {
	ID          int64  `json:"id"`
	Lan         string `json:"lan"`
	LanDoc      string `json:"lan_doc"`
	SubtitleUrl string `json:"subtitle_url"`
}

type BilibiliSubtitleInfo struct {
	AllowSubmit  bool                       `json:"allow_submit"`
	SubtitleList []BilibiliSubtitleProperty `json:"subtitles"`
}

type BilibiliWebInterfaceData struct {
	Bvid         string               `json:"bvid"`
	SubtitleInfo BilibiliSubtitleInfo `json:"subtitle"`
}

type BilibiliWebInterface struct {
	Code int                      `json:"code"`
	Data BilibiliWebInterfaceData `json:"data"`
}

type BilibiliFestival struct {
	VideoSections []struct {
		Id    int64  `json:"id"`
		Title string `json:"title"`
		Type  int    `json:"type"`
	} `json:"videoSections"`
	Episodes  []BilibiliEpisode `json:"episodes"`
	VideoInfo struct {
		Aid   int    `json:"aid"`
		BVid  string `json:"bvid"`
		Cid   int    `json:"cid"`
		Title string `json:"title"`
		Desc  string `json:"desc"`
		Pages []struct {
			Cid       int    `json:"cid"`
			Duration  int    `json:"duration"`
			Page      int    `json:"page"`
			Part      string `json:"part"`
			Dimension struct {
				Width  int `json:"width"`
				Height int `json:"height"`
				Rotate int `json:"rotate"`
			} `json:"dimension"`
		} `json:"pages"`
	} `json:"videoInfo"`
}
