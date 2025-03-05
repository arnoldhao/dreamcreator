package types

import (
	"github.com/asticode/go-astisub"
)

type ExtractorData struct {
	ID                string                                   `json:"id"`       // unique id
	Source            string                                   `json:"source"`   // website source name: youtube, bilibili, etc
	Site              string                                   `json:"site"`     // website name: youtube.com, bilibili.com, etc
	URL               string                                   `json:"url"`      // source video url
	Title             string                                   `json:"title"`    // source video title
	Streams           map[string]*ExtractorStream              `json:"streams"`  // each stream has it's own Parts and Quality
	Captions          map[string]*ExtractorCaption             `json:"captions"` // subtitles
	Danmakus          map[string]*ExtractorPart                `json:"danmakus"` // danmaku
	CaptionsTransform func([]byte) (*astisub.Subtitles, error) `json:"-"`        // transform caption to astisub format
	Err               error                                    `json:"err"`      // Err is used to record whether an error occurred when extracting the list data
}

type ExtractorCaption struct {
	LanguageCode string `json:"languageCode"`
	URL          string `json:"url"`
	Ext          string `json:"ext"`
}

// Part is the data structure for a single part of the video stream information.
type ExtractorPart struct {
	ID       string `json:"id"`
	FileName string `json:"fileName"`
	URL      string `json:"url"`
	Size     int64  `json:"size"`
	Ext      string `json:"ext"`
}

// Stream is the data structure for each video stream, eg: 720P, 1080P.
type ExtractorStream struct {
	// eg: "1080"
	ID string `json:"id"`
	// eg: "1080P xxx"
	Quality string `json:"quality"`
	// [Part: {URL, Size, Ext}, ...]
	// Some video stream have multiple parts,
	// and can also be used to download multiple image files at once
	Parts []*ExtractorPart `json:"parts"`
	// total size of all urls
	Size int64 `json:"size"`
	// the file extension after video parts merged
	Ext string `json:"ext"`
	// if the parts need mux
	NeedMux bool
}

// DataType indicates the type of extracted data, eg: video or image.
type ExtractorDataType string

const (
	// DataTypeAll indicates the type of extracted data is all.
	ExtractorDataTypeAll ExtractorDataType = "all"
	// DataTypeVideo indicates the type of extracted data is the video.
	ExtractorDataTypeVideo ExtractorDataType = "video"
	// DataTypeImage indicates the type of extracted data is the image.
	ExtractorDataTypeImage ExtractorDataType = "image"
	// DataTypeAudio indicates the type of extracted data is the audio.
	ExtractorDataTypeAudio ExtractorDataType = "audio"
	// DataTypeCaption indicates the type of extracted data is the caption.
	ExtractorDataTypeCaption ExtractorDataType = "caption"
)

// FillUpStreamsData fills up some data automatically.
func (d *ExtractorData) FillUpStreamsData() {
	if d.Streams == nil || len(d.Streams) == 0 {
		return
	}

	// 用于存储需要删除的 stream ID
	streamsToRemove := []string{}

	for id, stream := range d.Streams {
		// fill up ID
		stream.ID = id
		if stream.Quality == "" {
			stream.Quality = id
		}

		// generate the merged file extension
		if stream.Ext == "" {
			ext := stream.Parts[0].Ext
			// The file extension in `Parts` is used as the merged file extension by default, except for the following formats
			switch ext {
			// ts and flv files should be merged into an mp4 file
			case "ts", "flv", "f4v":
				ext = "mp4"
			}
			stream.Ext = ext
		}

		// calculate total size
		if stream.Size > 0 {
			continue
		}
		var size int64
		for _, part := range stream.Parts {
			size += part.Size
		}
		stream.Size = size

		// 如果 size 为 0，将该 stream 添加到待删除列表
		if size == 0 {
			streamsToRemove = append(streamsToRemove, id)
		}
	}

	// 从 d.Streams 中删除 size 为 0 的 stream
	for _, id := range streamsToRemove {
		delete(d.Streams, id)
	}
}

// ExtractorEmptyData returns an "empty" Extractor	Data object with the given URL and error.
func ExtractorEmptyData(url string, err error) *ExtractorData {
	return &ExtractorData{
		URL: url,
		Err: err,
	}
}

// ExtractorOptions defines optional options that can be used in the extraction function.
type ExtractorOptions struct {
	// Playlist indicates if we need to extract the whole playlist rather than the single video.
	Playlist bool
	// Items defines wanted items from a playlist. Separated by commas like: 1,5,6,8-10.
	Items string
	// ItemStart defines the starting item of a playlist.
	ItemStart int
	// ItemEnd defines the ending item of a playlist.
	ItemEnd int

	// ThreadNumber defines how many threads will use in the extraction, only works when Playlist is true.
	ThreadNumber int
	Cookie       string

	// EpisodeTitleOnly indicates file name of each bilibili episode doesn't include the playlist title
	EpisodeTitleOnly bool
}
