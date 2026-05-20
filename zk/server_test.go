package zk

import (
	"reflect"
	"testing"
)

func TestParentPaths(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"/", nil},
		{"/a", nil},
		{"/a/b", []string{"/a"}},
		{"/a/b/c", []string{"/a", "/a/b"}},
		{"a/b/c", []string{"/a", "/a/b"}},
		{"/a/b/c/d", []string{"/a", "/a/b", "/a/b/c"}},
	}
	for _, tt := range tests {
		got := parentPaths(tt.in)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("parentPaths(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}
