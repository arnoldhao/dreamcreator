package zhconvert

import (
	"fmt"
)

const (
	ZHCONVERT_API_URL = "https://api.zhconvert.org"
	ZHCONVERT_WEB_URL = "https://www.zhconvert.org"
)

type ConverterType int

const (
	// 从 1001 开始
	ZH_CONVERTER_SIMPLIFIED       ConverterType = iota + 1001 // 1001 简体化
	ZH_CONVERTER_TRADITIONAL                                  // 1002 繁体化
	ZH_CONVERTER_CHINA                                        // 1003 中国化
	ZH_CONVERTER_HONGKONG                                     // 1004 香港化
	ZH_CONVERTER_TAIWAN                                       // 1005 台湾化
	ZH_CONVERTER_PINYIN                                       // 1006 拼音化
	ZH_CONVERTER_BOPOMOFO                                     // 1007 注音化
	ZH_CONVERTER_MARS                                         // 1008 火星化
	ZH_CONVERTER_WIKI_SIMPLIFIED                              // 1009 维基简体化
	ZH_CONVERTER_WIKI_TRADITIONAL                             // 1010 维基繁体化
)

// String 返回转换器类型的字符串表示
func (c ConverterType) String() string {
	switch c {
	case ZH_CONVERTER_SIMPLIFIED:
		return "Simplified"
	case ZH_CONVERTER_TRADITIONAL:
		return "Traditional"
	case ZH_CONVERTER_CHINA:
		return "China"
	case ZH_CONVERTER_HONGKONG:
		return "Hongkong"
	case ZH_CONVERTER_TAIWAN:
		return "Taiwan"
	case ZH_CONVERTER_PINYIN:
		return "Pinyin"
	case ZH_CONVERTER_BOPOMOFO:
		return "Bopomofo"
	case ZH_CONVERTER_MARS:
		return "Mars"
	case ZH_CONVERTER_WIKI_SIMPLIFIED:
		return "WikiSimplified"
	case ZH_CONVERTER_WIKI_TRADITIONAL:
		return "WikiTraditional"
	default:
		return "Unknown"
	}
}

// IsValid 检查转换器类型是否有效
func (c ConverterType) IsValid() bool {
	return c >= ZH_CONVERTER_SIMPLIFIED && c <= ZH_CONVERTER_WIKI_TRADITIONAL
}

// FromString 从字符串创建转换器类型
func ConverterTypeFromString(s string) (ConverterType, error) {
	switch s {
	case "Simplified":
		return ZH_CONVERTER_SIMPLIFIED, nil
	case "Traditional":
		return ZH_CONVERTER_TRADITIONAL, nil
	case "China":
		return ZH_CONVERTER_CHINA, nil
	case "Hongkong":
		return ZH_CONVERTER_HONGKONG, nil
	case "Taiwan":
		return ZH_CONVERTER_TAIWAN, nil
	case "Pinyin":
		return ZH_CONVERTER_PINYIN, nil
	case "Bopomofo":
		return ZH_CONVERTER_BOPOMOFO, nil
	case "Mars":
		return ZH_CONVERTER_MARS, nil
	case "WikiSimplified":
		return ZH_CONVERTER_WIKI_SIMPLIFIED, nil
	case "WikiTraditional":
		return ZH_CONVERTER_WIKI_TRADITIONAL, nil
	default:
		return 0, fmt.Errorf("unknown converter type: %s", s)
	}
}
