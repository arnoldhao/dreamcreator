package youtube

import (
	"CanMe/backend/pkg/specials/proxy"
	"CanMe/backend/types"
	"CanMe/backend/utils/timeUtil"
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/asticode/go-astisub"
	"github.com/kkdai/youtube/v2"
)

func ParseCaptions(body []byte) (*astisub.Subtitles, error) {
	// check format
	var transcipt types.YoutubeTranscript
	astCaptions := astisub.NewSubtitles()
	if err := xml.Unmarshal(body, &transcipt); err == nil {
		for idx, t := range transcipt.Text {
			st := &astisub.Item{}
			st.Index = idx + 1
			if st.StartAt, err = timeUtil.ParseYoutubeTranscript(t.Start); err != nil {
				err = fmt.Errorf("ParseYoutubeTranscript: line %d: parsing srt duration %s failed: %w", idx, t.Start, err)
				return nil, err
			}
			if st.EndAt, err = timeUtil.ParseYoutubeTranscript(t.Start, t.Dur); err != nil {
				err = fmt.Errorf("ParseYoutubeTranscript: line %d: parsing srt duration %s failed: %w", idx, t.Start+t.Dur, err)
				return nil, err
			}

			st.Lines = append(st.Lines, astisub.Line{Items: []astisub.LineItem{{Text: strings.TrimSpace(t.Text)}}})
			astCaptions.Items = append(astCaptions.Items, st)
		}
	} else {
		var caption types.YoutubeCaption
		if err := xml.Unmarshal(body, &caption); err != nil {
			return nil, err
		}

		for idx, p := range caption.Body.P {
			st := &astisub.Item{}
			st.Index = idx + 1
			st.StartAt = time.Duration(p.T) * time.Millisecond
			st.EndAt = st.StartAt + time.Duration(p.D)*time.Millisecond

			st.Lines = append(st.Lines, astisub.Line{Items: []astisub.LineItem{{Text: strings.TrimSpace(p.Text)}}})
			astCaptions.Items = append(astCaptions.Items, st)
		}
	}
	return astCaptions, nil
}

func client() *youtube.Client {
	return &youtube.Client{
		HTTPClient: proxy.GetInstance().GetDefaultClient(),
	}
}
