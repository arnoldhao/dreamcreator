package types

import "encoding/xml"

type YoutubeCaption struct {
	XMLName xml.Name `xml:"timedtext" json:"timedtext,omitempty"`
	Text    string   `xml:",chardata" json:"text,omitempty"`
	Format  string   `xml:"format,attr" json:"format,omitempty"`
	Body    struct {
		Text string `xml:",chardata" json:"text,omitempty"`
		P    []struct {
			Text string `xml:",chardata" json:"text,omitempty"`
			T    int    `xml:"t,attr" json:"t,omitempty"`
			D    int    `xml:"d,attr" json:"d,omitempty"`
		} `xml:"p" json:"p,omitempty"`
	} `xml:"body" json:"body,omitempty"`
}

type YoutubeTranscript struct {
	XMLName  xml.Name `xml:"transcript" json:"transcript,omitempty"`
	Chardata string   `xml:",chardata" json:"chardata,omitempty"`
	Text     []struct {
		Text  string `xml:",chardata" json:"text,omitempty"`
		Start string `xml:"start,attr" json:"start,omitempty"`
		Dur   string `xml:"dur,attr" json:"dur,omitempty"`
	} `xml:"text" json:"text,omitempty"`
}
