package export

import "sort"

func AddToStringIndex(index map[string][]int, key string, value int) {
	if key == "" {
		return
	}
	index[key] = append(index[key], value)
}

func SortEventIndex(index map[string][]int, eventStarts map[int]int64) {
	for key := range index {
		sort.SliceStable(index[key], func(i, j int) bool {
			a := index[key][i]
			b := index[key][j]
			if eventStarts[a] != eventStarts[b] {
				return eventStarts[a] < eventStarts[b]
			}
			return a < b
		})
	}
}
