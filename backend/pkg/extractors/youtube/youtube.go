package youtube

import (
	"fmt"
	"slices"
	"strconv"

	"CanMe/backend/pkg/extractors"
	"CanMe/backend/pkg/request"
	"CanMe/backend/types"
	"CanMe/backend/utils/domainUtil"
	"CanMe/backend/utils/poolUtil"

	"github.com/kkdai/youtube/v2"
	"github.com/pkg/errors"
)

const (
	referer = "https://www.youtube.com"
	source  = "youtube"
	site    = "youtube.com"
)

func init() {
	e := New()
	extractors.Register(source, e)
	extractors.Register("youtu", e) // youtu.be
}

type extractor struct {
	client *youtube.Client
}

// New returns a youtube extractor.
func New() extractors.Extractor {
	return &extractor{
		client: client(),
	}
}

// Extract is the main function to extract the data.
func (e *extractor) Extract(url string, option types.ExtractorOptions) ([]*types.ExtractorData, error) {
	if !option.Playlist {
		video, err := e.client.GetVideo(url)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return []*types.ExtractorData{e.youtubeDownload(url, video)}, nil
	}

	playlist, err := e.client.GetPlaylist(url)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	needDownloadItems := extractors.NeedDownloadList(option.Items, option.ItemStart, option.ItemEnd, len(playlist.Videos))
	extractedData := make([]*types.ExtractorData, len(needDownloadItems))
	wgp := poolUtil.NewWaitGroupPool(option.ThreadNumber)
	dataIndex := 0
	for index, videoEntry := range playlist.Videos {
		if !slices.Contains(needDownloadItems, index+1) {
			continue
		}

		wgp.Add()
		go func(index int, entry *youtube.PlaylistEntry, extractedData []*types.ExtractorData) {
			defer wgp.Done()
			video, err := e.client.VideoFromPlaylistEntry(entry)
			if err != nil {
				return
			}
			extractedData[index] = e.youtubeDownload(url, video)
		}(dataIndex, videoEntry, extractedData)
		dataIndex++
	}
	wgp.Wait()
	return extractedData, nil
}

// youtubeDownload download function for single url
func (e *extractor) youtubeDownload(url string, video *youtube.Video) *types.ExtractorData {
	streams := make(map[string]*types.ExtractorStream, len(video.Formats))
	audioCache := make(map[string]*types.ExtractorPart)
	captions := make(map[string]*types.ExtractorCaption)

	// captions
	if len(video.CaptionTracks) > 0 {
		for _, caption := range video.CaptionTracks {
			captions[caption.LanguageCode] = &types.ExtractorCaption{
				LanguageCode: caption.LanguageCode,
				URL:          caption.BaseURL,
			}
		}
	}

	for i := range video.Formats {
		f := &video.Formats[i]
		itag := strconv.Itoa(f.ItagNo)
		quality := f.MimeType
		if f.QualityLabel != "" {
			quality = fmt.Sprintf("%s %s", f.QualityLabel, f.MimeType)
		}

		part, err := e.genPartByFormat(video, f)
		if err != nil {
			return types.ExtractorEmptyData(url, err)
		}
		stream := &types.ExtractorStream{
			ID:      itag,
			Parts:   []*types.ExtractorPart{part},
			Quality: quality,
			Ext:     part.Ext,
			NeedMux: true,
		}

		if f.AudioChannels == 0 {
			audioPart, ok := audioCache[part.Ext]
			if !ok {
				audio, err := getVideoAudio(video, part.Ext)
				if err != nil {
					return types.ExtractorEmptyData(url, err)
				}
				audioPart, err = e.genPartByFormat(video, audio)
				if err != nil {
					return types.ExtractorEmptyData(url, err)
				}
				audioCache[part.Ext] = audioPart
			}
			stream.Parts = append(stream.Parts, audioPart)
		}
		streams[itag] = stream
	}

	return &types.ExtractorData{
		Source:            source,
		Site:              site,
		Title:             video.Title,
		Streams:           streams,
		Captions:          captions,
		CaptionsTransform: ParseCaptions,
		URL:               url,
	}
}

func (e *extractor) genPartByFormat(video *youtube.Video, f *youtube.Format) (*types.ExtractorPart, error) {
	ext := getStreamExt(f.MimeType)
	url, err := e.client.GetStreamURL(video, f)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	size := f.ContentLength
	if size == 0 {
		size, _ = request.FastNew().Size(url, referer)
	}
	return &types.ExtractorPart{
		URL:  url,
		Size: size,
		Ext:  ext,
	}, nil
}

func getVideoAudio(v *youtube.Video, mimeType string) (*youtube.Format, error) {
	audioFormats := v.Formats.Type(mimeType).Type("audio")
	if len(audioFormats) == 0 {
		return nil, errors.New("no audio format found after filtering")
	}
	audioFormats.Sort()
	return &audioFormats[0], nil
}

func getStreamExt(streamType string) string {
	exts := domainUtil.MatchOneOf(streamType, `(\w+)/(\w+);`)
	if exts == nil || len(exts) < 3 {
		return ""
	}
	return exts[2]
}
