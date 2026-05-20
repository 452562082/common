// Package config loads a strongly-typed config struct from up to three sources
// applied in order of increasing precedence:
//
//  1. Defaults baked into the destination struct.
//  2. A YAML file (path given via WithFile).
//  3. Environment variables (path-derived: foo.bar.baz → FOO_BAR_BAZ by default).
//
// The destination must be a non-nil pointer to a struct. Field-tag conventions:
//
//	type Cfg struct {
//	    Listen  string        `yaml:"listen" env:"LISTEN" default:":8080"`
//	    Timeout time.Duration `yaml:"timeout" env:"TIMEOUT" default:"5s"`
//	    DB struct {
//	        DSN string `yaml:"dsn" env:"DSN"`
//	    } `yaml:"db" envPrefix:"DB_"`
//	}
//
// Tag semantics:
//
//   - `yaml:"..."`        — name in the YAML file (defaults to lowercased field name).
//   - `env:"..."`         — env-var name (omit to skip env override; "-" also skips).
//   - `default:"..."`     — string-encoded default, applied before YAML / env.
//   - `envPrefix:"..."`   — on struct fields, prefix applied to env names of children.
//
// The package is intentionally small (~250 lines, no external deps beyond
// gopkg.in/yaml.v3). For more elaborate needs (live reload, multiple file
// formats, remote sources) consider viper or koanf.
package config

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Options tunes Load behaviour.
type Options struct {
	// File is the path to a YAML file. Empty = skip the file step.
	File string

	// FileOptional, when true, suppresses errors from a missing File.
	FileOptional bool

	// EnvPrefix is prepended to every derived env-var name.
	// Example: EnvPrefix="MYAPP" + field env tag "PORT" → MYAPP_PORT.
	EnvPrefix string

	// DisableEnv skips the env-var application step.
	DisableEnv bool
}

// Load reads sources (defaults → file → env) into dst.
// dst must be a non-nil pointer to a struct.
func Load(dst any, opts Options) error {
	rv := reflect.ValueOf(dst)
	if rv.Kind() != reflect.Pointer || rv.IsNil() || rv.Elem().Kind() != reflect.Struct {
		return errors.New("config: dst must be a non-nil pointer to a struct")
	}

	if err := applyDefaults(rv.Elem()); err != nil {
		return fmt.Errorf("config: defaults: %w", err)
	}

	if opts.File != "" {
		if err := applyFile(dst, opts); err != nil {
			return err
		}
	}

	if !opts.DisableEnv {
		if err := applyEnv(rv.Elem(), opts.EnvPrefix); err != nil {
			return fmt.Errorf("config: env: %w", err)
		}
	}
	return nil
}

func applyFile(dst any, opts Options) error {
	data, err := os.ReadFile(opts.File)
	if err != nil {
		if os.IsNotExist(err) && opts.FileOptional {
			return nil
		}
		return fmt.Errorf("config: read %s: %w", opts.File, err)
	}
	dec := yaml.NewDecoder(strings.NewReader(string(data)))
	dec.KnownFields(true)
	if err := dec.Decode(dst); err != nil {
		return fmt.Errorf("config: decode %s: %w", opts.File, err)
	}
	return nil
}

// applyDefaults walks the struct, setting fields that carry a `default:"..."`
// tag and are still at their zero value.
func applyDefaults(v reflect.Value) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Struct {
			if err := applyDefaults(fv); err != nil {
				return err
			}
			continue
		}
		if fv.Kind() == reflect.Pointer && fv.Elem().Kind() == reflect.Struct {
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			if err := applyDefaults(fv.Elem()); err != nil {
				return err
			}
			continue
		}
		def, ok := f.Tag.Lookup("default")
		if !ok || def == "" {
			continue
		}
		if !fv.IsZero() {
			continue
		}
		if err := setField(fv, def); err != nil {
			return fmt.Errorf("field %s: %w", f.Name, err)
		}
	}
	return nil
}

// applyEnv walks the struct, overriding any field whose env-derived name is
// set in the process environment.
func applyEnv(v reflect.Value, prefix string) error {
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		fv := v.Field(i)
		if fv.Kind() == reflect.Struct {
			subPrefix := prefix
			if p, ok := f.Tag.Lookup("envPrefix"); ok && p != "" {
				subPrefix = joinPrefix(prefix, p)
			}
			if err := applyEnv(fv, subPrefix); err != nil {
				return err
			}
			continue
		}
		if fv.Kind() == reflect.Pointer && fv.Elem().Kind() == reflect.Struct {
			subPrefix := prefix
			if p, ok := f.Tag.Lookup("envPrefix"); ok && p != "" {
				subPrefix = joinPrefix(prefix, p)
			}
			if err := applyEnv(fv.Elem(), subPrefix); err != nil {
				return err
			}
			continue
		}
		env, ok := f.Tag.Lookup("env")
		if !ok || env == "-" || env == "" {
			continue
		}
		full := joinPrefix(prefix, env)
		raw, present := os.LookupEnv(full)
		if !present {
			continue
		}
		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("env %s: %w", full, err)
		}
	}
	return nil
}

func joinPrefix(parts ...string) string {
	out := ""
	for _, p := range parts {
		if p == "" {
			continue
		}
		if out == "" {
			out = p
			continue
		}
		if !strings.HasSuffix(out, "_") && !strings.HasPrefix(p, "_") {
			out += "_"
		}
		out += p
	}
	return out
}

// setField parses raw and assigns it to fv. The set of supported types covers
// what shows up in 95 % of config files; for anything exotic, leave the field
// out of the env/default story and populate it manually.
func setField(fv reflect.Value, raw string) error {
	if !fv.CanSet() {
		return errors.New("field is not settable")
	}
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("parse bool %q: %w", raw, err)
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// time.Duration is an Int64 under the hood; handle it specially.
		if fv.Type() == reflect.TypeOf(time.Duration(0)) {
			d, err := time.ParseDuration(raw)
			if err != nil {
				return fmt.Errorf("parse duration %q: %w", raw, err)
			}
			fv.SetInt(int64(d))
			return nil
		}
		n, err := strconv.ParseInt(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse int %q: %w", raw, err)
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse uint %q: %w", raw, err)
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, fv.Type().Bits())
		if err != nil {
			return fmt.Errorf("parse float %q: %w", raw, err)
		}
		fv.SetFloat(f)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element type %s", fv.Type().Elem().Kind())
		}
		parts := strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == ';' })
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if s := strings.TrimSpace(p); s != "" {
				out = append(out, s)
			}
		}
		fv.Set(reflect.ValueOf(out))
	default:
		return fmt.Errorf("unsupported field kind %s", fv.Kind())
	}
	return nil
}
