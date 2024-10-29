package capcut

import (
	"CanMe/backend/types"
	timeutil "CanMe/backend/utils/timeUtil"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/asticode/go-astisub"
)

type Capcut struct{}

func (p *Capcut) Format(ctx context.Context, fileName, jsonData string) (captions *astisub.Subtitles, err error) {
	if jsonData == "" {
		return nil, errors.New("jsonData is empty")
	}

	// unmarshal json
	var data types.CapCutContent
	err = json.Unmarshal([]byte(jsonData), &data)
	if err != nil {
		return nil, err
	}

	// check track
	if len(data.Tracks) == 0 {
		return nil, errors.New("no tracks found")
	}

	tracks := make(map[string]types.TargetTimerange)
	for _, track := range data.Tracks {
		if track.Segments != nil {
			for _, s := range track.Segments {
				tracks[s.MaterialID] = types.TargetTimerange{
					Duration: s.TargetTimerange.Duration,
					Start:    s.TargetTimerange.Start,
				}
			}
		}
	}

	// capcut -> astisub.Subtitles
	captions = astisub.NewSubtitles()
	for idx, text := range data.Materials.Texts {
		st := &astisub.Item{}
		if _, ok := tracks[text.ID]; !ok {
			fmt.Printf("Warning: No matching segment for text index %d\n", idx)
			continue
		}

		st.Index = idx + 1

		// generate time
		timeRange := tracks[text.ID]
		if st.StartAt, err = timeutil.ParseCapcut(timeRange.Start); err != nil {
			return nil, err
		}
		if st.EndAt, err = timeutil.ParseCapcut(timeRange.Start + timeRange.Duration); err != nil {
			return nil, err
		}

		// capcut subtitle text
		var content types.CapCutContentMaterialsTextsContent
		var subText string
		if err := json.Unmarshal([]byte(text.Content), &content); err == nil {
			subText = content.Text
		}
		st.Lines = append(st.Lines, astisub.Line{Items: []astisub.LineItem{{Text: strings.TrimSpace(subText)}}})

		// append to captions
		captions.Items = append(captions.Items, st)
	}

	return captions, nil
}
