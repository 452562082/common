package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

type subCfg struct {
	DSN     string `yaml:"dsn" env:"DSN" default:"postgres://localhost/dev"`
	MaxConn int    `yaml:"max_conn" env:"MAX_CONN" default:"10"`
}

type rootCfg struct {
	Listen  string        `yaml:"listen" env:"LISTEN" default:":8080"`
	Timeout time.Duration `yaml:"timeout" env:"TIMEOUT" default:"5s"`
	Debug   bool          `yaml:"debug" env:"DEBUG" default:"false"`
	Hosts   []string      `yaml:"hosts" env:"HOSTS"`
	DB      subCfg        `yaml:"db" envPrefix:"DB_"`
	Skipped string        `yaml:"-" env:"-"`
}

func TestLoad_DefaultsOnly(t *testing.T) {
	var c rootCfg
	if err := Load(&c, Options{}); err != nil {
		t.Fatal(err)
	}
	want := rootCfg{
		Listen:  ":8080",
		Timeout: 5 * time.Second,
		Debug:   false,
		DB:      subCfg{DSN: "postgres://localhost/dev", MaxConn: 10},
	}
	if !reflect.DeepEqual(c, want) {
		t.Errorf("got %+v, want %+v", c, want)
	}
}

func TestLoad_YAMLOverridesDefaults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	yaml := `
listen: ":9000"
timeout: 10s
debug: true
hosts:
  - h1
  - h2
db:
  dsn: postgres://h/p
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}

	var c rootCfg
	if err := Load(&c, Options{File: path}); err != nil {
		t.Fatal(err)
	}
	if c.Listen != ":9000" {
		t.Errorf("Listen = %q", c.Listen)
	}
	if c.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v", c.Timeout)
	}
	if !c.Debug {
		t.Errorf("Debug should be true")
	}
	if !reflect.DeepEqual(c.Hosts, []string{"h1", "h2"}) {
		t.Errorf("Hosts = %v", c.Hosts)
	}
	if c.DB.DSN != "postgres://h/p" {
		t.Errorf("DB.DSN = %q", c.DB.DSN)
	}
	if c.DB.MaxConn != 10 { // default kept (not in YAML)
		t.Errorf("DB.MaxConn = %d, expected default 10", c.DB.MaxConn)
	}
}

func TestLoad_EnvOverridesYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	_ = os.WriteFile(path, []byte("listen: \":7000\"\n"), 0o600)

	t.Setenv("APP_LISTEN", ":6000")
	t.Setenv("APP_HOSTS", "a,b;c")
	t.Setenv("APP_DB_MAX_CONN", "50")

	var c rootCfg
	if err := Load(&c, Options{File: path, EnvPrefix: "APP"}); err != nil {
		t.Fatal(err)
	}
	if c.Listen != ":6000" {
		t.Errorf("env should win: Listen = %q", c.Listen)
	}
	if !reflect.DeepEqual(c.Hosts, []string{"a", "b", "c"}) {
		t.Errorf("Hosts = %v", c.Hosts)
	}
	if c.DB.MaxConn != 50 {
		t.Errorf("nested env override failed: %d", c.DB.MaxConn)
	}
}

func TestLoad_MissingFileOptional(t *testing.T) {
	var c rootCfg
	err := Load(&c, Options{File: "/nope/missing.yaml", FileOptional: true})
	if err != nil {
		t.Fatalf("FileOptional should swallow missing: %v", err)
	}
	if c.Listen != ":8080" {
		t.Errorf("defaults should still apply: %q", c.Listen)
	}
}

func TestLoad_MissingFileRequired(t *testing.T) {
	var c rootCfg
	err := Load(&c, Options{File: "/nope/missing.yaml"})
	if err == nil {
		t.Fatal("expected error when file missing and not optional")
	}
}

func TestLoad_DSTValidation(t *testing.T) {
	if err := Load(nil, Options{}); err == nil {
		t.Error("nil should fail")
	}
	var s rootCfg
	if err := Load(s, Options{}); err == nil {
		t.Error("non-pointer should fail")
	}
}

func TestLoad_UnknownYAMLField(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.yaml")
	_ = os.WriteFile(path, []byte("listen: \":1\"\nbogus: 42\n"), 0o600)

	var c rootCfg
	if err := Load(&c, Options{File: path}); err == nil {
		t.Error("KnownFields should reject 'bogus'")
	}
}

func TestSetField_Duration(t *testing.T) {
	var c struct {
		D time.Duration `env:"D"`
	}
	t.Setenv("D", "750ms")
	if err := Load(&c, Options{DisableEnv: false}); err != nil {
		t.Fatal(err)
	}
	if c.D != 750*time.Millisecond {
		t.Errorf("got %v", c.D)
	}
}
