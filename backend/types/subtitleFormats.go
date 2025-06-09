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
	Head struct {
		Metadata struct {
			Title     string `xml:"title"`
			Copyright string `xml:"copyright"`
			Language  string `xml:"language"`
			TcFormat  string `xml:"tcFormat"`
		} `xml:"metadata"`
		Styling struct {
			Styles []struct {
				ID              string  `xml:"id,attr"`
				FontFamily      string  `xml:"fontFamily"`
				FontSize        float64 `xml:"fontSize"`
				Color           string  `xml:"color"`
				TextAlign       string  `xml:"textAlign"`
				BackgroundColor string  `xml:"backgroundColor"`
			} `xml:"style"`
		} `xml:"styling"`
	} `xml:"head"`
	Body struct {
		Div []struct {
			P []struct {
				Begin   string `xml:"begin,attr"`
				End     string `xml:"end,attr"`
				Style   string `xml:"style,attr"`
				Speaker string `xml:"speaker,attr"`
				Content string `xml:",chardata"`
			} `xml:"p"`
		} `xml:"div"`
	} `xml:"body"`
}
