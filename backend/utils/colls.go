package utils

type CollsVoid struct{}

// Set 集合, 存放不重复的元素
type CollsSet[T Hashable] map[T]CollsVoid

func CollsNewSet[T Hashable](elems ...T) CollsSet[T] {
	if len(elems) > 0 {
		data := make(CollsSet[T], len(elems))
		for _, e := range elems {
			data[e] = CollsVoid{}
		}
		return data
	} else {
		return CollsSet[T]{}
	}
}
