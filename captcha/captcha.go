// Package captcha generates simple image-based CAPTCHAs.
//
// The package draws the answer as wavy text on an image with random noise
// pixels and dots, then PNG-encodes it. There's no third-party image
// dependency — we use stdlib image / image/png / image/draw only.
//
// Persistence is pluggable via Store: ship with an in-memory store; production
// deployments should wire a Redis-backed store so multiple replicas share
// pending challenges.
//
// Usage:
//
//	cap := captcha.New(captcha.Options{
//	    Width: 160, Height: 60, Length: 5,
//	    TTL: 5 * time.Minute,
//	    Store: captcha.NewMemoryStore(),
//	})
//
//	id, png, _ := cap.Generate(ctx)            // give to client
//	ok, _ := cap.Verify(ctx, id, userAnswer)   // single-use check
package captcha

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	mrand "math/rand"
	"strings"
	"sync"
	"time"
)

// Options configures New.
type Options struct {
	// Width / Height of the rendered image. Defaults: 160 × 60.
	Width  int
	Height int

	// Length of the answer string. Default 5.
	Length int

	// Charset to draw from. Default: "0123456789ABCDEFGHJKMNPQRSTUVWXYZ"
	// (digits + uppercase, minus visually-ambiguous characters).
	Charset string

	// TTL is how long an issued challenge is valid. Default 5 minutes.
	TTL time.Duration

	// CaseSensitive controls Verify's comparison. Default false (case-insensitive).
	CaseSensitive bool

	// Store provides issuance persistence. nil ⇒ NewMemoryStore.
	Store Store
}

// Store is the persistence interface for issued challenges. Implement this
// with Redis if you need it shared across replicas.
type Store interface {
	Put(ctx context.Context, id, answer string, ttl time.Duration) error
	// Take retrieves and atomically deletes the entry (single-use semantics).
	// Returns ("", false, nil) when the id is unknown / expired.
	Take(ctx context.Context, id string) (string, bool, error)
}

// Captcha is the high-level helper.
type Captcha struct {
	opts Options
}

// New builds a Captcha with opts applied.
func New(opts Options) *Captcha {
	if opts.Width == 0 {
		opts.Width = 160
	}
	if opts.Height == 0 {
		opts.Height = 60
	}
	if opts.Length == 0 {
		opts.Length = 5
	}
	if opts.Charset == "" {
		opts.Charset = "0123456789ABCDEFGHJKMNPQRSTUVWXYZ"
	}
	if opts.TTL == 0 {
		opts.TTL = 5 * time.Minute
	}
	if opts.Store == nil {
		opts.Store = NewMemoryStore()
	}
	return &Captcha{opts: opts}
}

// Generate creates a new challenge, persists it, and returns (id, pngImage).
func (c *Captcha) Generate(ctx context.Context) (id string, png []byte, err error) {
	answer := c.randomAnswer()
	id, err = randomID()
	if err != nil {
		return "", nil, fmt.Errorf("captcha: gen id: %w", err)
	}
	if err := c.opts.Store.Put(ctx, id, answer, c.opts.TTL); err != nil {
		return "", nil, fmt.Errorf("captcha: store put: %w", err)
	}
	img := c.draw(answer)
	var buf bytes.Buffer
	if err := pngEncode(&buf, img); err != nil {
		return "", nil, fmt.Errorf("captcha: encode: %w", err)
	}
	return id, buf.Bytes(), nil
}

// Verify checks the user's answer against the stored challenge. Single-use:
// success or not, the challenge is consumed and the next Verify(id, ...) on
// the same id returns (false, nil).
//
// The comparison is constant-time. Timing differences on short captcha
// answers are negligible in practice, but constant-time matching also
// eliminates the lower-noise side channel of length mismatch and is a
// good default for any "is this token / secret correct?" predicate.
func (c *Captcha) Verify(ctx context.Context, id, userAnswer string) (bool, error) {
	answer, ok, err := c.opts.Store.Take(ctx, id)
	if err != nil {
		return false, fmt.Errorf("captcha: store take: %w", err)
	}
	if !ok {
		return false, nil
	}
	a, u := answer, userAnswer
	if !c.opts.CaseSensitive {
		a = strings.ToLower(a)
		u = strings.ToLower(u)
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(u)) == 1, nil
}

// ---------- Drawing ---------------------------------------------------------

// randomAnswer draws c.opts.Length characters from the configured charset
// using crypto/rand. Captcha answers MUST be unpredictable — using math/rand
// here would let an attacker recover the PRNG state from a handful of issued
// answers and predict every subsequent challenge.
func (c *Captcha) randomAnswer() string {
	chars := c.opts.Charset
	n := len(chars)
	out := make([]byte, c.opts.Length)

	// Read enough bytes in one syscall, then sample uniformly via modular
	// rejection so we don't bias toward early characters.
	buf := make([]byte, c.opts.Length*2+8)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand failure on a production system is fatal; fall back
		// to math/rand only to avoid panics, but emit zeros if that fails too.
		for i := range out {
			out[i] = chars[mrand.Intn(n)]
		}
		return string(out)
	}

	// Largest multiple of n that fits in a uint16 — rejection threshold.
	const maxU16 = 1 << 16
	threshold := maxU16 - (maxU16 % n)

	bi := 0
	for i := 0; i < len(out); {
		if bi+2 > len(buf) {
			// Top up the buffer.
			if _, err := rand.Read(buf); err != nil {
				out[i] = chars[mrand.Intn(n)]
				i++
				continue
			}
			bi = 0
		}
		r := int(buf[bi]) | int(buf[bi+1])<<8
		bi += 2
		if r >= threshold {
			continue
		}
		out[i] = chars[r%n]
		i++
	}
	return string(out)
}

func (c *Captcha) draw(answer string) image.Image {
	w, h := c.opts.Width, c.opts.Height
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	// Background: very light grey.
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.RGBA{R: 240, G: 240, B: 240, A: 255}}, image.Point{}, draw.Src)

	// Noise dots.
	for range (w * h) / 20 {
		img.Set(mrand.Intn(w), mrand.Intn(h), randomDarkColor())
	}
	// Distractor curves.
	for range 3 {
		drawSineCurve(img, randomDarkColor())
	}
	// Letters: a 5x7 raster font, scaled and slightly rotated/offset per glyph.
	x0 := max(4, (w-len(answer)*charBoxW)/2)
	for i, r := range answer {
		drawChar(img, x0+i*charBoxW, (h-charBoxH)/2+mrand.Intn(6)-3, byte(r))
	}
	return img
}

// pngEncode is a tiny wrapper so test code can replace the encoder if needed.
func pngEncode(buf *bytes.Buffer, img image.Image) error {
	return png.Encode(buf, img)
}

func randomDarkColor() color.RGBA {
	return color.RGBA{
		R: uint8(mrand.Intn(120)),
		G: uint8(mrand.Intn(120)),
		B: uint8(mrand.Intn(120)),
		A: 255,
	}
}

func drawSineCurve(img *image.RGBA, c color.RGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	amp := float64(h) / 4
	period := float64(w) / (2 + mrand.Float64()*2)
	yOff := float64(h)/2 + (mrand.Float64()*float64(h)/3 - float64(h)/6)
	for x := range w {
		y := int(amp*math.Sin(float64(x)/period) + yOff)
		if y >= 0 && y < h {
			img.Set(x, y, c)
			if y+1 < h {
				img.Set(x, y+1, c)
			}
		}
	}
}

// ---------- Memory store ----------------------------------------------------

// MemoryStore is an in-process Store. Safe for concurrent use; not shared
// across replicas. Useful for tests / single-instance deployments.
//
// It runs a background goroutine that periodically evicts expired entries
// so that abandoned (never-verified) captchas don't pile up in memory —
// captcha endpoints are typically unauthenticated, so this matters.
type MemoryStore struct {
	mu      sync.Mutex
	entries map[string]memEntry

	stop chan struct{}
	done chan struct{}
}

type memEntry struct {
	answer  string
	expires time.Time
}

// NewMemoryStore returns an empty in-memory store with background eviction
// running at a sensible default interval (1 minute). Call Close to stop it.
func NewMemoryStore() *MemoryStore {
	return NewMemoryStoreWithInterval(time.Minute)
}

// NewMemoryStoreWithInterval is like NewMemoryStore but lets the caller pick
// the sweep cadence. interval <= 0 disables the background goroutine — only
// do this in tests.
func NewMemoryStoreWithInterval(interval time.Duration) *MemoryStore {
	s := &MemoryStore{
		entries: make(map[string]memEntry),
		stop:    make(chan struct{}),
		done:    make(chan struct{}),
	}
	if interval > 0 {
		go s.sweepLoop(interval)
	} else {
		close(s.done)
	}
	return s
}

// Put adds an entry. Expired entries are not removed at Put time; the
// background sweeper handles that.
func (s *MemoryStore) Put(_ context.Context, id, answer string, ttl time.Duration) error {
	s.mu.Lock()
	s.entries[id] = memEntry{answer: answer, expires: time.Now().Add(ttl)}
	s.mu.Unlock()
	return nil
}

// Take retrieves & deletes id atomically.
func (s *MemoryStore) Take(_ context.Context, id string) (string, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok {
		return "", false, nil
	}
	delete(s.entries, id)
	if time.Now().After(e.expires) {
		return "", false, nil
	}
	return e.answer, true, nil
}

// Len reports the number of pending entries (for tests).
func (s *MemoryStore) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

// Close stops the background sweeper. Safe to call multiple times.
func (s *MemoryStore) Close() error {
	select {
	case <-s.stop:
		return nil
	default:
	}
	close(s.stop)
	<-s.done
	return nil
}

func (s *MemoryStore) sweepLoop(interval time.Duration) {
	defer close(s.done)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-t.C:
			s.evictExpired()
		}
	}
}

func (s *MemoryStore) evictExpired() {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for k, e := range s.entries {
		if now.After(e.expires) {
			delete(s.entries, k)
		}
	}
}

// ---------- IDs --------------------------------------------------------------

func randomID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	const hex = "0123456789abcdef"
	out := make([]byte, len(b)*2)
	for i, v := range b {
		out[i*2] = hex[v>>4]
		out[i*2+1] = hex[v&0x0f]
	}
	return string(out), nil
}

// ErrIDInvalid is reserved for future store implementations that need to
// distinguish "malformed id" from "id not found". Currently unused.
var ErrIDInvalid = errors.New("captcha: invalid id")
