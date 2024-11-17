package extractors

import (
	"errors"
	"strconv"
	"strings"
)

var (
	// ErrURLParseFailed defines url parse failed error.
	ErrURLParseFailed            = errors.New("url parse failed")
	ErrInvalidRegularExpression  = errors.New("invalid regular expression")
	ErrURLQueryParamsParseFailed = errors.New("url query params parse failed")
	ErrBodyParseFailed           = errors.New("body parse failed")
)

// NeedDownloadList return the indices of playlist that need download
func NeedDownloadList(items string, itemStart, itemEnd, length int) []int {
	if items != "" {
		var itemList []int
		var selStart, selEnd int
		temp := strings.Split(items, ",")

		for _, i := range temp {
			selection := strings.Split(i, "-")
			selStart, _ = strconv.Atoi(strings.TrimSpace(selection[0]))

			if len(selection) >= 2 {
				selEnd, _ = strconv.Atoi(strings.TrimSpace(selection[1]))
			} else {
				selEnd = selStart
			}

			for item := selStart; item <= selEnd; item++ {
				itemList = append(itemList, item)
			}
		}
		return itemList
	}

	if itemStart < 1 {
		itemStart = 1
	}
	if itemEnd == 0 {
		itemEnd = length
	}
	if itemEnd < itemStart {
		itemEnd = itemStart
	}
	return Range(itemStart, itemEnd)
}

// Range generate a sequence of numbers by range
func Range(min, max int) []int {
	items := make([]int, max-min+1)
	for index := range items {
		items[index] = min + index
	}
	return items
}
