package utils

// Map map items to new array
func SliceMap[S ~[]T, T any, R any](arr S, mappingFunc func(int) R) []R {
	total := len(arr)
	result := make([]R, total)
	for i := 0; i < total; i++ {
		result[i] = mappingFunc(i)
	}
	return result
}
