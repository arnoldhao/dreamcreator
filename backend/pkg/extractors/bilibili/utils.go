package bilibili

import (
	"encoding/json"
	"fmt"

	"CanMe/backend/types"
	"CanMe/backend/utils/timeUtil"

	"github.com/asticode/go-astisub"
)

func ParseCaptions(body []byte) (*astisub.Subtitles, error) {
	captionData := types.BilibiliSubtitleFormat{}
	err := json.Unmarshal(body, &captionData)
	if err != nil {
		return nil, err
	}

	astCaptions := astisub.NewSubtitles()
	for i := 0; i < len(captionData.Body); i++ {
		st := &astisub.Item{}
		st.Index = i
		if st.StartAt, err = timeUtil.ParseBilibiliSubtitle(captionData.Body[i].From); err != nil {
			err = fmt.Errorf("ParseBilibiliSubtitle: line %d: parsing srt duration %.2f failed: %w", i, captionData.Body[i].From, err)
			return nil, err
		}
		if st.EndAt, err = timeUtil.ParseBilibiliSubtitle(captionData.Body[i].To); err != nil {
			err = fmt.Errorf("ParseBilibiliSubtitle: line %d: parsing srt duration %.2f failed: %w", i, captionData.Body[i].To, err)
			return nil, err
		}
		st.Lines = append(st.Lines, astisub.Line{Items: []astisub.LineItem{{Text: captionData.Body[i].Content}}})
		astCaptions.Items = append(astCaptions.Items, st)
	}

	return astCaptions, nil
}
