package util

import (
	"reflect"
	"testing"
)

func TestChunkSlice(t *testing.T) {
	tests := []struct {
		name     string
		items    []int
		n        int
		expected [][]int
	}{
		{name: "empty slice", items: []int{}, n: 4, expected: nil},
		{name: "nil slice", items: nil, n: 1, expected: nil},
		{name: "single chunk", items: []int{1, 2, 3}, n: 1, expected: [][]int{{1, 2, 3}}},
		{name: "even split", items: []int{1, 2, 3, 4}, n: 2, expected: [][]int{{1, 2}, {3, 4}}},
		{name: "uneven split", items: []int{1, 2, 3, 4, 5}, n: 2, expected: [][]int{{1, 2, 3}, {4, 5}}},
		// More workers than items used to panic with "slice bounds out of range".
		{name: "more chunks than items", items: []int{1}, n: 4, expected: [][]int{{1}}},
		{name: "more chunks than items, two items", items: []int{1, 2}, n: 4, expected: [][]int{{1}, {2}}},
		{name: "more chunks than items, five items", items: []int{1, 2, 3, 4, 5}, n: 4, expected: [][]int{{1, 2}, {3, 4}, {5}}},
		// Zero used to panic with "integer divide by zero".
		{name: "zero chunks", items: []int{1, 2}, n: 0, expected: [][]int{{1, 2}}},
		{name: "negative chunks", items: []int{1, 2}, n: -3, expected: [][]int{{1, 2}}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := ChunkSlice(test.items, test.n)
			if !reflect.DeepEqual(got, test.expected) {
				t.Fatalf("ChunkSlice(%v, %d) = %v, want %v", test.items, test.n, got, test.expected)
			}
		})
	}
}

func TestChunkSliceKeepsEveryItemExactlyOnce(t *testing.T) {
	items := make([]int, 17)
	for i := range items {
		items[i] = i
	}

	for n := -2; n <= 20; n++ {
		var flat []int
		for _, chunk := range ChunkSlice(items, n) {
			if len(chunk) == 0 {
				t.Fatalf("n=%d produced an empty chunk", n)
			}
			flat = append(flat, chunk...)
		}
		if !reflect.DeepEqual(flat, items) {
			t.Fatalf("n=%d lost or reordered items: %v", n, flat)
		}
	}
}
