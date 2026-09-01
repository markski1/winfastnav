package ui

import "testing"

func TestFirstResultSelectedWhileTyping(t *testing.T) {
	tests := []struct {
		query       string
		resultCount int
		want        bool
	}{
		{query: "calc", resultCount: 2, want: true},
		{query: "  calc  ", resultCount: 1, want: true},
		{query: "", resultCount: 2, want: false},
		{query: "   ", resultCount: 2, want: false},
		{query: "calc", resultCount: 0, want: false},
	}

	for _, test := range tests {
		if got := firstResultSelected(test.query, test.resultCount); got != test.want {
			t.Errorf("firstResultSelected(%q, %d) = %v, want %v", test.query, test.resultCount, got, test.want)
		}
	}
}
