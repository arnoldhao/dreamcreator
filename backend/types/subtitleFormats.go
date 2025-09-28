// 创建专门的FCPXML结构体文件
package types

import "encoding/xml"

// FCPXML 根元素
type FCPXML struct {
	XMLName   xml.Name         `xml:"fcpxml"`
	Version   string           `xml:"version,attr"`
	Resources *FCPXMLResources `xml:"resources,omitempty"`
	Library   *FCPXMLLibrary   `xml:"library,omitempty"`
}

// Resources 资源定义
type FCPXMLResources struct {
	XMLName xml.Name       `xml:"resources"`
	Formats []FCPXMLFormat `xml:"format,omitempty"`
	Effects []FCPXMLEffect `xml:"effect,omitempty"`
}

// Format 格式定义
type FCPXMLFormat struct {
	XMLName       xml.Name `xml:"format"`
	ID            string   `xml:"id,attr"`
	Name          string   `xml:"name,attr"`
	FrameDuration string   `xml:"frameDuration,attr"`
	Width         int      `xml:"width,attr"`
	Height        int      `xml:"height,attr"`
	ColorSpace    string   `xml:"colorSpace,attr"`
}

// Effect 效果定义
type FCPXMLEffect struct {
	XMLName xml.Name `xml:"effect"`
	ID      string   `xml:"id,attr"`
	Name    string   `xml:"name,attr"`
	UID     string   `xml:"uid,attr"`
}

// Library 库定义
type FCPXMLLibrary struct {
	XMLName  xml.Name      `xml:"library"`
	Location string        `xml:"location,attr"`
	Events   []FCPXMLEvent `xml:"event,omitempty"`
}

// Event 事件定义
type FCPXMLEvent struct {
	XMLName  xml.Name        `xml:"event"`
	Name     string          `xml:"name,attr"`
	UID      string          `xml:"uid,attr"`
	Projects []FCPXMLProject `xml:"project,omitempty"`
}

// Project 项目定义
type FCPXMLProject struct {
	XMLName  xml.Name        `xml:"project"`
	Name     string          `xml:"name,attr"`
	UID      string          `xml:"uid,attr"`
	ModDate  string          `xml:"modDate,attr"`
	Sequence *FCPXMLSequence `xml:"sequence,omitempty"`
}

// Sequence 序列定义
type FCPXMLSequence struct {
	XMLName     xml.Name     `xml:"sequence"`
	Duration    string       `xml:"duration,attr"`
	Format      string       `xml:"format,attr"`
	TCStart     string       `xml:"tcStart,attr"`
	TCFormat    string       `xml:"tcFormat,attr"`
	AudioLayout string       `xml:"audioLayout,attr"`
	AudioRate   string       `xml:"audioRate,attr"`
	Spine       *FCPXMLSpine `xml:"spine,omitempty"`
}

// Spine 主轨道
type FCPXMLSpine struct {
	XMLName xml.Name          `xml:"spine"`
	Items   []FCPXMLSpineItem `xml:",any"`
}

// SpineItem 轨道项目接口
type FCPXMLSpineItem interface{}

// Gap 空隙
type FCPXMLGap struct {
	XMLName  xml.Name      `xml:"gap"`
	Name     string        `xml:"name,attr"`
	Offset   string        `xml:"offset,attr"`
	Duration string        `xml:"duration,attr"`
	Start    string        `xml:"start,attr"`
	Titles   []FCPXMLTitle `xml:"title,omitempty"`
}

// Title 标题
type FCPXMLTitle struct {
	XMLName      xml.Name             `xml:"title"`
	Name         string               `xml:"name,attr"`
	Lane         int                  `xml:"lane,attr"`
	Offset       string               `xml:"offset,attr"`
	Ref          string               `xml:"ref,attr,omitempty"`
	Duration     string               `xml:"duration,attr"`
	Start        string               `xml:"start,attr"`
	Params       []FCPXMLParam        `xml:"param,omitempty"`
	Text         *FCPXMLText          `xml:"text,omitempty"`
	TextStyleDef []FCPXMLTextStyleDef `xml:"text-style-def,omitempty"`
}

// Param 参数
type FCPXMLParam struct {
	XMLName xml.Name `xml:"param"`
	Name    string   `xml:"name,attr"`
	Key     string   `xml:"key,attr"`
	Value   string   `xml:"value,attr"`
}

// Text 文本内容
type FCPXMLText struct {
	XMLName   xml.Name          `xml:"text"`
	TextStyle []FCPXMLTextStyle `xml:"text-style,omitempty"`
}

// TextStyle 文本样式
type FCPXMLTextStyle struct {
	XMLName xml.Name `xml:"text-style"`
	Ref     string   `xml:"ref,attr,omitempty"`
	Content string   `xml:",chardata"`
}

// TextStyleDef 文本样式定义
type FCPXMLTextStyleDef struct {
	XMLName   xml.Name             `xml:"text-style-def"`
	ID        string               `xml:"id,attr"`
	TextStyle *FCPXMLTextStyleAttr `xml:"text-style,omitempty"`
}

// TextStyleAttr 文本样式属性
type FCPXMLTextStyleAttr struct {
	XMLName      xml.Name `xml:"text-style"`
	Font         string   `xml:"font,attr,omitempty"`
	FontSize     string   `xml:"fontSize,attr,omitempty"`
	FontFace     string   `xml:"fontFace,attr,omitempty"`
	FontColor    string   `xml:"fontColor,attr,omitempty"`
	Bold         string   `xml:"bold,attr,omitempty"`
	ShadowColor  string   `xml:"shadowColor,attr,omitempty"`
	ShadowOffset string   `xml:"shadowOffset,attr,omitempty"`
	Alignment    string   `xml:"alignment,attr,omitempty"`
}

// IttDocument itt格式字幕
type IttDocument struct {
	XMLName xml.Name `xml:"tt"`
	// Namespaces and parameters per TTML/iTT	n
	Xmlns                  string `xml:"xmlns,attr,omitempty"`
	XmlnsTtp               string `xml:"xmlns:ttp,attr,omitempty"`
	XmlnsTts               string `xml:"xmlns:tts,attr,omitempty"`
	XmlnsTtm               string `xml:"xmlns:ttm,attr,omitempty"`
	XmlLang                string `xml:"xml:lang,attr,omitempty"`
	TtpTimeBase            string `xml:"timeBase,attr,omitempty"`
	TtpFrameRate           string `xml:"frameRate,attr,omitempty"`
	TtpFrameRateMultiplier string `xml:"frameRateMultiplier,attr,omitempty"`
	TtpDropMode            string `xml:"dropMode,attr,omitempty"`
	Head                   struct {
		Metadata struct {
			Title     string `xml:"title,omitempty"`
			Copyright string `xml:"copyright,omitempty"`
			Tool      string `xml:"agent,omitempty"`
		} `xml:"metadata"`
		Styling struct {
			Styles []struct {
				XMLID         string `xml:"xml:id,attr,omitempty"`
				TtsFontFamily string `xml:"fontFamily,attr,omitempty"`
				TtsFontSize   string `xml:"fontSize,attr,omitempty"`
				TtsTextAlign  string `xml:"textAlign,attr,omitempty"`
				TtsColor      string `xml:"color,attr,omitempty"`
				TtsBackground string `xml:"backgroundColor,attr,omitempty"`
				TtsFontStyle  string `xml:"fontStyle,attr,omitempty"`
				TtsFontWeight string `xml:"fontWeight,attr,omitempty"`
			} `xml:"style"`
		} `xml:"styling"`
		Layout struct {
			Regions []struct {
				XMLID           string `xml:"xml:id,attr"`
				TtsOrigin       string `xml:"origin,attr,omitempty"`
				TtsExtent       string `xml:"extent,attr,omitempty"`
				TtsDisplayAlign string `xml:"displayAlign,attr,omitempty"`
				TtsWritingMode  string `xml:"writingMode,attr,omitempty"`
			} `xml:"region"`
		} `xml:"layout"`
	} `xml:"head"`
	Body struct {
		Style    string `xml:"style,attr,omitempty"`
		Region   string `xml:"region,attr,omitempty"`
		TtsColor string `xml:"color,attr,omitempty"`
		Div      []struct {
			P []struct {
				Begin   string `xml:"begin,attr"`
				End     string `xml:"end,attr"`
				Region  string `xml:"region,attr,omitempty"`
				Style   string `xml:"style,attr,omitempty"`
				Speaker string `xml:"speaker,attr,omitempty"`
				Content string `xml:",chardata"`
			} `xml:"p"`
		} `xml:"div"`
	} `xml:"body"`
}
