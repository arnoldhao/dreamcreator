package bilibili

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"CanMe/backend/pkg/extractors"
	"CanMe/backend/pkg/parser"
	"CanMe/backend/pkg/request"
	"CanMe/backend/types"
	"CanMe/backend/utils/domainUtil"
	"CanMe/backend/utils/poolUtil"
)

func init() {
	bilibiliExtractor := New()
	extractors.Register("bilibili", bilibiliExtractor)
	extractors.Register("b23", bilibiliExtractor)
}

const (
	bilibiliAPI        = "https://api.bilibili.com/x/player/playurl?"
	bilibiliBangumiAPI = "https://api.bilibili.com/pgc/player/web/playurl?"
	bilibiliTokenAPI   = "https://api.bilibili.com/x/player/playurl/token?"
)

const (
	referer = "https://www.bilibili.com"
	source  = "bilibili"
	site    = "bilibili.com"
)

var utoken string

func genAPI(aid, cid, quality int, bvid string, bangumi bool, cookie string) (string, error) {
	var (
		err        error
		baseAPIURL string
		params     string
	)
	if cookie != "" && utoken == "" {
		utoken, err = request.FastNew().Get(
			fmt.Sprintf("%said=%d&cid=%d", bilibiliTokenAPI, aid, cid),
			referer,
			nil,
		)
		if err != nil {
			return "", err
		}
		var t types.BilibiliToken
		err = json.Unmarshal([]byte(utoken), &t)
		if err != nil {
			return "", err
		}
		if t.Code != 0 {
			return "", errors.Errorf("cookie error: %s", t.Message)
		}
		utoken = t.Data.Token
	}
	var api string
	if bangumi {
		// The parameters need to be sorted by name
		// qn=0 flag makes the CDN address different every time
		// quality=120(4k) is the highest quality so far
		params = fmt.Sprintf(
			"cid=%d&bvid=%s&qn=%d&type=&otype=json&fourk=1&fnver=0&fnval=16",
			cid, bvid, quality,
		)
		baseAPIURL = bilibiliBangumiAPI
	} else {
		params = fmt.Sprintf(
			"avid=%d&cid=%d&bvid=%s&qn=%d&type=&otype=json&fourk=1&fnver=0&fnval=2000",
			aid, cid, bvid, quality,
		)
		baseAPIURL = bilibiliAPI
	}
	api = baseAPIURL + params
	// bangumi utoken also need to put in params to sign, but the ordinary video doesn't need
	if !bangumi && utoken != "" {
		api = fmt.Sprintf("%s&utoken=%s", api, utoken)
	}
	return api, nil
}

type bilibiliOptions struct {
	url      string
	html     string
	bangumi  bool
	aid      int
	cid      int
	bvid     string
	page     int
	subtitle string
}

func extractBangumi(url, html string, extractOption types.ExtractorOptions) ([]*types.ExtractorData, error) {
	dataString := domainUtil.MatchOneOf(html, `<script\s+id="__NEXT_DATA__"\s+type="application/json"\s*>(.*?)</script\s*>`)[1]
	epArrayString := domainUtil.MatchOneOf(dataString, `"episode_info"\s*:\s*(.+?)\s*,\s*"season_info"`)[1]
	fullVideoIdString := domainUtil.MatchOneOf(dataString, `"videoId"\s*:\s*"(ep|ss)(\d+)"`)
	epSsString := fullVideoIdString[1] // "ep" or "ss"
	videoIdString := fullVideoIdString[2]

	var epArray types.BilibiliEpVideoInfo
	err := json.Unmarshal([]byte(epArrayString), &epArray)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var data types.BilibiliBangumiData

	videoId, err := strconv.ParseInt(videoIdString, 10, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if epArray.EpID == int(videoId) || (epSsString == "ss" && epArray.Title == "第1话") {
		data.EpInfo = epArray
	}
	data.EpList = append(data.EpList, epArray)

	sort.Slice(data.EpList, func(i, j int) bool {
		return data.EpList[i].EpID < data.EpList[j].EpID
	})

	if !extractOption.Playlist {
		aid := data.EpInfo.Aid
		cid := data.EpInfo.Cid
		bvid := data.EpInfo.Bvid
		titleFormat := data.EpInfo.Title
		longTitle := data.EpInfo.LongTitle
		if aid <= 0 || cid <= 0 || bvid == "" {
			aid = data.EpList[0].Aid
			cid = data.EpList[0].Cid
			bvid = data.EpList[0].Bvid
			titleFormat = data.EpList[0].Title
			longTitle = data.EpList[0].LongTitle
		}
		options := bilibiliOptions{
			url:     url,
			html:    html,
			bangumi: true,
			aid:     aid,
			cid:     cid,
			bvid:    bvid,

			subtitle: fmt.Sprintf("%s %s", titleFormat, longTitle),
		}
		return []*types.ExtractorData{bilibiliDownload(options, extractOption)}, nil
	}

	// handle bangumi playlist
	needDownloadItems := extractors.NeedDownloadList(extractOption.Items, extractOption.ItemStart, extractOption.ItemEnd, len(data.EpList))
	extractedData := make([]*types.ExtractorData, len(needDownloadItems))
	wgp := poolUtil.NewWaitGroupPool(extractOption.ThreadNumber)
	dataIndex := 0
	for index, u := range data.EpList {
		if !slices.Contains(needDownloadItems, index+1) {
			continue
		}
		wgp.Add()
		id := u.EpID
		if id == 0 {
			id = u.EpID
		}
		// html content can't be reused here
		options := bilibiliOptions{
			url:     fmt.Sprintf("https://www.bilibili.com/bangumi/play/ep%d", id),
			bangumi: true,
			aid:     u.Aid,
			cid:     u.Cid,
			bvid:    u.Bvid,

			subtitle: fmt.Sprintf("%s %s", u.Title, u.LongTitle),
		}
		go func(index int, options bilibiliOptions, extractedData []*types.ExtractorData) {
			defer wgp.Done()
			extractedData[index] = bilibiliDownload(options, extractOption)
		}(dataIndex, options, extractedData)
		dataIndex++
	}
	wgp.Wait()
	return extractedData, nil
}

func getMultiPageData(html string) (*types.BilibiliMultiPage, error) {
	var data types.BilibiliMultiPage
	multiPageDataString := domainUtil.MatchOneOf(
		html, `window.__INITIAL_STATE__=(.+?);\(function`,
	)
	if multiPageDataString == nil {
		return &data, errors.New("this page has no playlist")
	}
	err := json.Unmarshal([]byte(multiPageDataString[1]), &data)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &data, nil
}

func extractFestival(url, html string, extractOption types.ExtractorOptions) ([]*types.ExtractorData, error) {
	matches := domainUtil.MatchAll(html, "<\\s*script[^>]*>\\s*window\\.__INITIAL_STATE__=([\\s\\S]*?);\\s?\\(function[\\s\\S]*?<\\/\\s*script\\s*>")
	if len(matches) < 1 {
		return nil, errors.WithStack(extractors.ErrURLParseFailed)
	}
	if len(matches[0]) < 2 {
		return nil, errors.New("could not find video in page")
	}

	var festivalData types.BilibiliFestival
	err := json.Unmarshal([]byte(matches[0][1]), &festivalData)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	options := bilibiliOptions{
		url:  url,
		html: html,
		aid:  festivalData.VideoInfo.Aid,
		bvid: festivalData.VideoInfo.BVid,
		cid:  festivalData.VideoInfo.Cid,
		page: 0,
	}

	return []*types.ExtractorData{bilibiliDownload(options, extractOption)}, nil
}

func extractNormalVideo(url, html string, extractOption types.ExtractorOptions) ([]*types.ExtractorData, error) {
	pageData, err := getMultiPageData(html)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if !extractOption.Playlist {
		// handle URL that has a playlist, mainly for unified titles
		// <h1> tag does not include subtitles
		// bangumi doesn't need this
		pageString := domainUtil.MatchOneOf(url, `\?p=(\d+)`)
		var p int
		if pageString == nil {
			// https://www.bilibili.com/video/av20827366/
			p = 1
		} else {
			// https://www.bilibili.com/video/av20827366/?p=2
			p, _ = strconv.Atoi(pageString[1])
		}

		if len(pageData.VideoData.Pages) < p || p < 1 {
			return nil, errors.WithStack(extractors.ErrURLParseFailed)
		}

		page := pageData.VideoData.Pages[p-1]
		options := bilibiliOptions{
			url:  url,
			html: html,
			aid:  pageData.Aid,
			bvid: pageData.BVid,
			cid:  page.Cid,
			page: p,
		}
		// "part":"" or "part":"Untitled"
		if page.Part == "Untitled" || len(pageData.VideoData.Pages) == 1 {
			options.subtitle = ""
		} else {
			options.subtitle = page.Part
		}
		return []*types.ExtractorData{bilibiliDownload(options, extractOption)}, nil
	}

	// handle normal video playlist
	if len(pageData.Sections) == 0 {
		// https://www.bilibili.com/video/av20827366/?p=* each video in playlist has different p=?
		return multiPageDownload(url, html, extractOption, pageData)
	}
	// handle another kind of playlist
	// https://www.bilibili.com/video/av*** each video in playlist has different av/bv id
	return multiEpisodeDownload(url, html, extractOption, pageData)
}

// handle multi episode download
func multiEpisodeDownload(url, html string, extractOption types.ExtractorOptions, pageData *types.BilibiliMultiPage) ([]*types.ExtractorData, error) {
	needDownloadItems := extractors.NeedDownloadList(extractOption.Items, extractOption.ItemStart, extractOption.ItemEnd, len(pageData.Sections[0].Episodes))
	extractedData := make([]*types.ExtractorData, len(needDownloadItems))
	wgp := poolUtil.NewWaitGroupPool(extractOption.ThreadNumber)
	dataIndex := 0
	for index, u := range pageData.Sections[0].Episodes {
		if !slices.Contains(needDownloadItems, index+1) {
			continue
		}
		wgp.Add()
		options := bilibiliOptions{
			url:      url,
			html:     html,
			aid:      u.Aid,
			bvid:     u.BVid,
			cid:      u.Cid,
			subtitle: fmt.Sprintf("%s P%d", u.Title, index+1),
		}
		go func(index int, options bilibiliOptions, extractedData []*types.ExtractorData) {
			defer wgp.Done()
			extractedData[index] = bilibiliDownload(options, extractOption)
		}(dataIndex, options, extractedData)
		dataIndex++
	}
	wgp.Wait()
	return extractedData, nil
}

// handle multi page download
func multiPageDownload(url, html string, extractOption types.ExtractorOptions, pageData *types.BilibiliMultiPage) ([]*types.ExtractorData, error) {
	needDownloadItems := extractors.NeedDownloadList(extractOption.Items, extractOption.ItemStart, extractOption.ItemEnd, len(pageData.VideoData.Pages))
	extractedData := make([]*types.ExtractorData, len(needDownloadItems))
	wgp := poolUtil.NewWaitGroupPool(extractOption.ThreadNumber)
	dataIndex := 0
	for index, u := range pageData.VideoData.Pages {
		if !slices.Contains(needDownloadItems, index+1) {
			continue
		}
		wgp.Add()
		options := bilibiliOptions{
			url:      url,
			html:     html,
			aid:      pageData.Aid,
			bvid:     pageData.BVid,
			cid:      u.Cid,
			subtitle: u.Part,
			page:     u.Page,
		}
		go func(index int, options bilibiliOptions, extractedData []*types.ExtractorData) {
			defer wgp.Done()
			extractedData[index] = bilibiliDownload(options, extractOption)
		}(dataIndex, options, extractedData)
		dataIndex++
	}
	wgp.Wait()
	return extractedData, nil
}

type extractor struct{}

// New returns a bilibili extractor.
func New() extractors.Extractor {
	return &extractor{}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, option types.ExtractorOptions) ([]*types.ExtractorData, error) {
	// Clean the URL first
	cleanURL, err := e.cleanBilibiliURL(url)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	html, err := request.FastNew().Get(cleanURL, referer, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// set thread number to 1 manually to avoid http 412 error
	option.ThreadNumber = 1

	if strings.Contains(cleanURL, "bangumi") {
		// handle bangumi
		return extractBangumi(cleanURL, html, option)
	} else if strings.Contains(cleanURL, "festival") {
		return extractFestival(cleanURL, html, option)
	} else {
		// handle normal video
		return extractNormalVideo(cleanURL, html, option)
	}
}

func (e *extractor) cleanBilibiliURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// Keep only the path which contains the video ID
	cleanURL := fmt.Sprintf("%s://%s%s", parsedURL.Scheme, parsedURL.Host, parsedURL.Path)
	return cleanURL, nil
}

// bilibiliDownload is the download function for a single URL
func bilibiliDownload(options bilibiliOptions, extractOption types.ExtractorOptions) *types.ExtractorData {
	var (
		err  error
		html string
	)
	if options.html != "" {
		// reuse html string, but this can't be reused in case of playlist
		html = options.html
	} else {
		html, err = request.FastNew().Get(options.url, referer, nil)
		if err != nil {
			return types.ExtractorEmptyData(options.url, err)
		}
	}

	// Get "accept_quality" and "accept_description"
	// "accept_description":["超高清 8K","超清 4K","高清 1080P+","高清 1080P","高清 720P","清晰 480P","流畅 360P"],
	// "accept_quality":[127，120,112,80,48,32,16],
	api, err := genAPI(options.aid, options.cid, 127, options.bvid, options.bangumi, extractOption.Cookie)
	if err != nil {
		return types.ExtractorEmptyData(options.url, err)
	}
	jsonString, err := request.FastNew().Get(api, referer, nil)
	if err != nil {
		return types.ExtractorEmptyData(options.url, err)
	}

	var data types.BilibiliDash
	err = json.Unmarshal([]byte(jsonString), &data)
	if err != nil {
		return types.ExtractorEmptyData(options.url, err)
	}
	var dashData types.BilibiliDashInfo
	if data.Data.Description == nil {
		dashData = data.Result
	} else {
		dashData = data.Data
	}

	var audioPart *types.ExtractorPart
	if dashData.Streams.Audio != nil {
		// Get audio part
		var audioID int
		audios := map[int]string{}
		bandwidth := 0
		for _, stream := range dashData.Streams.Audio {
			if stream.Bandwidth > bandwidth {
				audioID = stream.ID
				bandwidth = stream.Bandwidth
			}
			audios[stream.ID] = stream.BaseURL
		}
		s, err := request.FastNew().Size(audios[audioID], referer)
		if err != nil {
			return types.ExtractorEmptyData(options.url, err)
		}
		audioPart = &types.ExtractorPart{
			URL:  audios[audioID],
			Size: s,
			Ext:  "m4a",
		}
	}

	streams := make(map[string]*types.ExtractorStream, len(dashData.Quality))
	for _, stream := range dashData.Streams.Video {
		s, err := request.FastNew().Size(stream.BaseURL, referer)
		if err != nil {
			return types.ExtractorEmptyData(options.url, err)
		}
		parts := make([]*types.ExtractorPart, 0, 2)
		parts = append(parts, &types.ExtractorPart{
			URL:  stream.BaseURL,
			Size: s,
			Ext:  getExtFromMimeType(stream.MimeType),
		})
		if audioPart != nil {
			parts = append(parts, audioPart)
		}
		var size int64
		for _, part := range parts {
			size += part.Size
		}
		id := fmt.Sprintf("%d-%d", stream.ID, stream.Codecid)
		streams[id] = &types.ExtractorStream{
			Parts:   parts,
			Size:    size,
			Quality: fmt.Sprintf("%s %s", types.BilibiliQualityString[stream.ID], stream.Codecs),
		}
		if audioPart != nil {
			streams[id].NeedMux = true
		}
	}

	for _, durl := range dashData.DURLs {
		var ext string
		switch dashData.DURLFormat {
		case "flv", "flv480":
			ext = "flv"
		case "mp4", "hdmp4": // nolint
			ext = "mp4"
		}

		parts := make([]*types.ExtractorPart, 0, 1)
		parts = append(parts, &types.ExtractorPart{
			URL:  durl.URL,
			Size: durl.Size,
			Ext:  ext,
		})

		streams[strconv.Itoa(dashData.CurQuality)] = &types.ExtractorStream{
			Parts:   parts,
			Size:    durl.Size,
			Quality: types.BilibiliQualityString[dashData.CurQuality],
		}
	}

	// get the title
	doc, err := parser.NewDocument(html)
	if err != nil {
		return types.ExtractorEmptyData(options.url, err)
	}
	title, err := doc.ExtractTitle()
	if err != nil {
		return types.ExtractorEmptyData(options.url, err)
	}
	if options.subtitle != "" {
		pageString := ""
		if options.page > 0 {
			pageString = fmt.Sprintf("P%d ", options.page)
		}
		if extractOption.EpisodeTitleOnly {
			title = fmt.Sprintf("%s%s", pageString, options.subtitle)
		} else {
			title = fmt.Sprintf("%s %s%s", title, pageString, options.subtitle)
		}
	}

	// danmakus
	danmakus := make(map[string]*types.ExtractorPart, 1)
	danmakus["danmaku"] = &types.ExtractorPart{
		URL: fmt.Sprintf("https://comment.bilibili.com/%d.xml", options.cid),
		Ext: "xml",
	}

	return &types.ExtractorData{
		Source:            source,
		Site:              site,
		Title:             title,
		Streams:           streams,
		Captions:          getSubTitleCaptionPart(options.aid, options.cid),
		CaptionsTransform: ParseCaptions,
		Danmakus:          danmakus,
		URL:               options.url,
	}
}

func getExtFromMimeType(mimeType string) string {
	exts := strings.Split(mimeType, "/")
	if len(exts) == 2 {
		return exts[1]
	}
	return "mp4"
}

func getSubTitleCaptionPart(aid int, cid int) map[string]*types.ExtractorCaption {
	jsonString, err := request.FastNew().Get(
		fmt.Sprintf("http://api.bilibili.com/x/player/wbi/v2?aid=%d&cid=%d", aid, cid), referer, nil,
	)
	if err != nil {
		return nil
	}
	stu := types.BilibiliWebInterface{}
	err = json.Unmarshal([]byte(jsonString), &stu)
	if err != nil || len(stu.Data.SubtitleInfo.SubtitleList) == 0 {
		return nil
	}

	captions := make(map[string]*types.ExtractorCaption, len(stu.Data.SubtitleInfo.SubtitleList))
	for _, v := range stu.Data.SubtitleInfo.SubtitleList {
		captions[v.Lan] = &types.ExtractorCaption{
			URL: fmt.Sprintf("https:%s", v.SubtitleUrl),
			Ext: "srt",
		}
	}

	return captions
}
