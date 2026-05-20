package env

import (
	"errors"
	"reflect"
	"testing"
)

func TestSplitHosts(t *testing.T) {
	tests := []struct {
		in   string
		want []string
	}{
		{"", nil},
		{"a", []string{"a"}},
		{"a,b,c", []string{"a", "b", "c"}},
		{"a;b;c", []string{"a", "b", "c"}},
		{"a, b;c", []string{"a", "b", "c"}},
		{"  a , , b ", []string{"a", "b"}},
		{",,;,", nil},
	}
	for _, tt := range tests {
		got := splitHosts(tt.in)
		if len(got) == 0 && len(tt.want) == 0 {
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("splitHosts(%q) = %v, want %v", tt.in, got, tt.want)
		}
	}
}

func TestGetHosts_Unset(t *testing.T) {
	t.Setenv("ZK_HOSTS", "")
	if _, err := ZooKeeperHosts(); !errors.Is(err, ErrUnset) {
		t.Fatalf("expected ErrUnset, got %v", err)
	}
}

func TestGetHosts_OK(t *testing.T) {
	t.Setenv("KAFKA_HOSTS", "host1:9092,host2:9092;host3:9092")
	got, err := KafkaHosts()
	if err != nil {
		t.Fatalf("KafkaHosts: %v", err)
	}
	want := []string{"host1:9092", "host2:9092", "host3:9092"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("KafkaHosts = %v, want %v", got, want)
	}
}

func TestGetString(t *testing.T) {
	t.Setenv("HOST_IP", "10.0.0.1")
	v, err := HostIP()
	if err != nil {
		t.Fatalf("HostIP: %v", err)
	}
	if v != "10.0.0.1" {
		t.Errorf("HostIP = %q, want %q", v, "10.0.0.1")
	}
}

func TestGetString_WhitespaceOnly(t *testing.T) {
	t.Setenv("SERVER_HOST", "   ")
	if _, err := ServerHost(); !errors.Is(err, ErrUnset) {
		t.Fatalf("expected ErrUnset, got %v", err)
	}
}
