package middleware

import (
	"net/url"
	"testing"
)

func TestRedactQuery_EmptyAllowReturnsEmpty(t *testing.T) {
	q := url.Values{"token": {"abc"}, "page": {"1"}}
	if got := redactQuery(q, map[string]struct{}{}); got != "" {
		t.Errorf("empty allow should drop the query; got %q", got)
	}
}

func TestRedactQuery_KeepsAllowedRedactsRest(t *testing.T) {
	q := url.Values{
		"token":  {"super-secret"},
		"page":   {"3"},
		"apikey": {"ABCD"},
		"sort":   {"name"},
	}
	allow := map[string]struct{}{"page": {}, "sort": {}}
	got := redactQuery(q, allow)
	want := "apikey=***&page=3&sort=name&token=***"
	if got != want {
		t.Errorf("redactQuery = %q, want %q", got, want)
	}
}

func TestRedactQuery_StableOrder(t *testing.T) {
	q := url.Values{"b": {"2"}, "a": {"1"}, "c": {"3"}}
	allow := map[string]struct{}{"a": {}, "b": {}, "c": {}}
	if got := redactQuery(q, allow); got != "a=1&b=2&c=3" {
		t.Errorf("ordering not stable: %q", got)
	}
}

func TestRedactQuery_MultiValue(t *testing.T) {
	q := url.Values{"tag": {"red", "blue"}}
	allow := map[string]struct{}{"tag": {}}
	if got := redactQuery(q, allow); got != "tag=red&tag=blue" {
		t.Errorf("multivalue: %q", got)
	}
}
