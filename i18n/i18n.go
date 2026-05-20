// Package i18n is a minimal multi-language translator.
//
// It uses a self-contained design (no third-party deps) suitable for
// straightforward message-catalog needs: load JSON/YAML files keyed by
// language tag, then look up messages with placeholder substitution. For
// plural rules and ICU MessageFormat use the (heavier) nicksnyder/go-i18n.
//
// Usage:
//
//	b := i18n.NewBundle(language.English)
//	_ = b.LoadJSONFile("locales/en.json", language.English)
//	_ = b.LoadJSONFile("locales/zh.json", language.SimplifiedChinese)
//
//	l := b.Localizer(language.SimplifiedChinese)
//	fmt.Println(l.T("welcome", i18n.Args{"name": "Ada"}))
//
// Catalog file format (JSON example):
//
//	{
//	  "welcome": "Hello, {{name}}!",
//	  "errors.not_found": "{{resource}} not found"
//	}
package i18n

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"os"
	"strings"
	"sync"

	"golang.org/x/text/language"
)

// Args holds placeholder substitutions for a translation lookup.
type Args map[string]any

// Bundle holds catalogs keyed by language tag, plus a default fallback.
type Bundle struct {
	mu       sync.RWMutex
	defaultT language.Tag
	matcher  language.Matcher
	catalogs map[language.Tag]map[string]string
	tags     []language.Tag
}

// NewBundle returns an empty Bundle with the given default language.
func NewBundle(defaultLang language.Tag) *Bundle {
	b := &Bundle{
		defaultT: defaultLang,
		catalogs: make(map[language.Tag]map[string]string),
	}
	b.rebuildMatcher()
	return b
}

func (b *Bundle) rebuildMatcher() {
	tags := make([]language.Tag, 0, len(b.catalogs)+1)
	tags = append(tags, b.defaultT)
	for t := range b.catalogs {
		if t == b.defaultT {
			continue
		}
		tags = append(tags, t)
	}
	b.tags = tags
	b.matcher = language.NewMatcher(tags)
}

// AddMessages registers a map of messages for a language.
func (b *Bundle) AddMessages(lang language.Tag, messages map[string]string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if _, ok := b.catalogs[lang]; !ok {
		b.catalogs[lang] = make(map[string]string, len(messages))
	}
	maps.Copy(b.catalogs[lang], messages)
	b.rebuildMatcher()
}

// LoadJSONFile reads a JSON map (key → template) and registers it under lang.
func (b *Bundle) LoadJSONFile(path string, lang language.Tag) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("i18n: read %s: %w", path, err)
	}
	var messages map[string]string
	if err := json.Unmarshal(data, &messages); err != nil {
		return fmt.Errorf("i18n: decode %s: %w", path, err)
	}
	b.AddMessages(lang, messages)
	return nil
}

// Languages returns the registered language tags (including the default).
func (b *Bundle) Languages() []language.Tag {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]language.Tag, len(b.tags))
	copy(out, b.tags)
	return out
}

// Localizer is the lookup handle for one specific language.
type Localizer struct {
	bundle *Bundle
	lang   language.Tag
}

// Localizer returns a Localizer for the closest match to the requested lang.
// Unknown / unloaded languages fall back to the bundle's default.
func (b *Bundle) Localizer(lang language.Tag) *Localizer {
	b.mu.RLock()
	_, idx, _ := b.matcher.Match(lang)
	tag := b.tags[idx]
	b.mu.RUnlock()
	return &Localizer{bundle: b, lang: tag}
}

// LocalizerFor parses one or more Accept-Language values (in BCP-47 syntax)
// and returns the best Localizer for them.
//
//	l := b.LocalizerFor(r.Header.Get("Accept-Language"))
func (b *Bundle) LocalizerFor(acceptLanguageHeader string) *Localizer {
	tags, _, err := language.ParseAcceptLanguage(acceptLanguageHeader)
	if err != nil || len(tags) == 0 {
		return &Localizer{bundle: b, lang: b.defaultT}
	}
	b.mu.RLock()
	_, idx, _ := b.matcher.Match(tags...)
	tag := b.tags[idx]
	b.mu.RUnlock()
	return &Localizer{bundle: b, lang: tag}
}

// Lang returns the resolved language tag for this Localizer.
func (l *Localizer) Lang() language.Tag { return l.lang }

// T returns the translated message for key. Placeholders of the form
// "{{name}}" are replaced by args["name"] (rendered via fmt.Sprint).
// Missing keys fall back to the bundle's default catalog, and ultimately
// to the raw key string if no catalog has it.
func (l *Localizer) T(key string, args ...Args) string {
	tmpl := l.lookup(key)
	if tmpl == "" {
		return key
	}
	if len(args) == 0 {
		return tmpl
	}
	return substitute(tmpl, args[0])
}

// Has reports whether the key is registered for this localizer's language
// (default fallback NOT considered).
func (l *Localizer) Has(key string) bool {
	l.bundle.mu.RLock()
	defer l.bundle.mu.RUnlock()
	cat, ok := l.bundle.catalogs[l.lang]
	if !ok {
		return false
	}
	_, ok = cat[key]
	return ok
}

func (l *Localizer) lookup(key string) string {
	l.bundle.mu.RLock()
	defer l.bundle.mu.RUnlock()
	if cat, ok := l.bundle.catalogs[l.lang]; ok {
		if v, ok := cat[key]; ok {
			return v
		}
	}
	if l.lang != l.bundle.defaultT {
		if cat, ok := l.bundle.catalogs[l.bundle.defaultT]; ok {
			if v, ok := cat[key]; ok {
				return v
			}
		}
	}
	return ""
}

func substitute(tmpl string, args Args) string {
	if len(args) == 0 || !strings.Contains(tmpl, "{{") {
		return tmpl
	}
	var sb strings.Builder
	sb.Grow(len(tmpl))
	for i := 0; i < len(tmpl); {
		if i+1 < len(tmpl) && tmpl[i] == '{' && tmpl[i+1] == '{' {
			end := strings.Index(tmpl[i+2:], "}}")
			if end < 0 {
				sb.WriteString(tmpl[i:])
				break
			}
			key := strings.TrimSpace(tmpl[i+2 : i+2+end])
			if v, ok := args[key]; ok {
				fmt.Fprint(&sb, v)
			} else {
				sb.WriteString(tmpl[i : i+2+end+2])
			}
			i += 2 + end + 2
			continue
		}
		sb.WriteByte(tmpl[i])
		i++
	}
	return sb.String()
}

// ---------- HTTP convenience -------------------------------------------------

// FromRequest is a one-liner: build a Localizer using r's Accept-Language header.
func (b *Bundle) FromRequest(r *http.Request) *Localizer {
	return b.LocalizerFor(r.Header.Get("Accept-Language"))
}

// ErrUnknownKey is returned by Must when a key is missing from every catalog.
var ErrUnknownKey = errors.New("i18n: unknown key")

// Must returns the translation or an error when the key is unknown anywhere.
func (l *Localizer) Must(key string, args ...Args) (string, error) {
	if l.lookup(key) == "" {
		return "", fmt.Errorf("%w: %s", ErrUnknownKey, key)
	}
	return l.T(key, args...), nil
}
