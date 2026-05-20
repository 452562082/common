package i18n

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/text/language"
)

func newTestBundle() *Bundle {
	b := NewBundle(language.English)
	b.AddMessages(language.English, map[string]string{
		"welcome":         "Hello, {{name}}!",
		"error.notfound":  "{{resource}} not found",
		"only.in.english": "english only",
	})
	b.AddMessages(language.SimplifiedChinese, map[string]string{
		"welcome":        "你好, {{name}}!",
		"error.notfound": "{{resource}} 未找到",
	})
	return b
}

func TestLocalizer_Basic(t *testing.T) {
	b := newTestBundle()
	en := b.Localizer(language.English)
	if got := en.T("welcome", Args{"name": "Ada"}); got != "Hello, Ada!" {
		t.Errorf("got %q", got)
	}
	zh := b.Localizer(language.SimplifiedChinese)
	if got := zh.T("welcome", Args{"name": "Ada"}); got != "你好, Ada!" {
		t.Errorf("got %q", got)
	}
}

func TestLocalizer_FallbackToDefault(t *testing.T) {
	b := newTestBundle()
	zh := b.Localizer(language.SimplifiedChinese)
	// Key only exists in the English catalog — should fall back.
	if got := zh.T("only.in.english"); got != "english only" {
		t.Errorf("expected fallback to default, got %q", got)
	}
}

func TestLocalizer_UnknownKeyReturnsKey(t *testing.T) {
	b := newTestBundle()
	if got := b.Localizer(language.English).T("nope.no.such"); got != "nope.no.such" {
		t.Errorf("got %q", got)
	}
}

func TestLocalizer_Has(t *testing.T) {
	b := newTestBundle()
	zh := b.Localizer(language.SimplifiedChinese)
	if !zh.Has("welcome") {
		t.Error("Has(welcome) should be true")
	}
	if zh.Has("only.in.english") {
		t.Error("Has should not see default-fallback keys")
	}
}

func TestLocalizer_Must(t *testing.T) {
	b := newTestBundle()
	if _, err := b.Localizer(language.English).Must("welcome", Args{"name": "X"}); err != nil {
		t.Errorf("known key should not error: %v", err)
	}
	if _, err := b.Localizer(language.English).Must("nope"); err == nil {
		t.Error("unknown key should error")
	}
}

func TestSubstitute_MissingArg(t *testing.T) {
	out := substitute("Hello {{name}}, you have {{count}}", Args{"name": "Ada"})
	if !strings.Contains(out, "Hello Ada") || !strings.Contains(out, "{{count}}") {
		t.Errorf("missing arg should be left in place; got %q", out)
	}
}

func TestSubstitute_NoPlaceholders(t *testing.T) {
	if got := substitute("plain text", Args{"x": "y"}); got != "plain text" {
		t.Errorf("got %q", got)
	}
}

func baseOf(t language.Tag) language.Base {
	b, _ := t.Base()
	return b
}

func TestLocalizerFor_AcceptLanguage(t *testing.T) {
	b := newTestBundle()
	l := b.LocalizerFor("zh-CN,en;q=0.8")
	if baseOf(l.Lang()) != baseOf(language.SimplifiedChinese) {
		t.Errorf("matched %v, expected zh-Hans", l.Lang())
	}

	// Unknown locale → falls back to default (English).
	l = b.LocalizerFor("xx-YY")
	if baseOf(l.Lang()) != baseOf(language.English) {
		t.Errorf("matched %v, expected default English", l.Lang())
	}

	// Garbage header → default.
	l = b.LocalizerFor("not a header")
	if baseOf(l.Lang()) != baseOf(language.English) {
		t.Errorf("garbage header should default")
	}
}

func TestBundle_FromRequest(t *testing.T) {
	b := newTestBundle()
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("Accept-Language", "zh-CN")
	l := b.FromRequest(r)
	if baseOf(l.Lang()) != baseOf(language.SimplifiedChinese) {
		t.Errorf("matched %v", l.Lang())
	}
}

func TestLoadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fr.json")
	_ = os.WriteFile(path, []byte(`{"welcome":"Bonjour, {{name}}!"}`), 0o600)

	b := NewBundle(language.English)
	if err := b.LoadJSONFile(path, language.French); err != nil {
		t.Fatal(err)
	}
	if got := b.Localizer(language.French).T("welcome", Args{"name": "Ada"}); got != "Bonjour, Ada!" {
		t.Errorf("got %q", got)
	}
}

func TestLoadJSONFile_BadFile(t *testing.T) {
	b := NewBundle(language.English)
	if err := b.LoadJSONFile("/no/such/file.json", language.German); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLanguages(t *testing.T) {
	b := newTestBundle()
	langs := b.Languages()
	if len(langs) < 2 {
		t.Errorf("expected at least 2 languages, got %v", langs)
	}
}
