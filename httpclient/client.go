// Package httpclient provides a small, dependency-free wrapper around
// net/http that adds:
//
//   - Sensible default timeouts and a tuned connection pool.
//   - Configurable per-request retries with exponential backoff and jitter.
//   - JSON helpers (GetJSON / PostJSON) for the common case.
//
// The returned *Client embeds *http.Client, so any standard net/http call
// site keeps working unchanged.
package httpclient

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Options configures New.
type Options struct {
	// Timeout is the maximum duration for the entire request (including
	// retries). Zero falls back to 30s. Use a per-request ctx for tighter
	// control of a single attempt.
	Timeout time.Duration

	// DialTimeout caps the TCP dial. Default 5s.
	DialTimeout time.Duration

	// KeepAlive controls TCP keep-alive. Default 30s.
	KeepAlive time.Duration

	// MaxIdleConns is the global cap on idle connections. Default 100.
	MaxIdleConns int

	// MaxIdleConnsPerHost limits per-host idle connections. Default 32.
	MaxIdleConnsPerHost int

	// IdleConnTimeout is how long an idle connection is kept open. Default 90s.
	IdleConnTimeout time.Duration

	// MaxRetries is the number of retry attempts after the first try.
	// Zero disables retries.
	MaxRetries int

	// RetryWaitMin / RetryWaitMax bracket the exponential backoff window.
	RetryWaitMin time.Duration
	RetryWaitMax time.Duration

	// RetryOn is consulted to decide whether to retry. If nil, DefaultRetryOn
	// is used (retry network errors and 5xx).
	RetryOn func(resp *http.Response, err error) bool

	// UserAgent, if non-empty, is sent on every request that doesn't already
	// set its own User-Agent header.
	UserAgent string

	// AllowHost, if non-nil, decides whether the host portion of a target
	// URL (and every redirect along the way) is reachable. Use this to harden
	// against SSRF when forwarding caller-supplied URLs — typically combined
	// with DenyPrivateIP. Returning false aborts the request with ErrHostNotAllowed.
	AllowHost func(host string) bool

	// MaxRedirects caps redirect chains. 0 falls back to net/http's default
	// of 10. Set to -1 to disallow any redirect.
	MaxRedirects int

	// Transport overrides the underlying http.RoundTripper. When set, all
	// connection-pool options above are ignored.
	Transport http.RoundTripper
}

// ErrHostNotAllowed is returned when AllowHost rejects a target.
var ErrHostNotAllowed = errors.New("httpclient: host not allowed")

// ErrTooManyRedirects is returned when MaxRedirects is exceeded.
var ErrTooManyRedirects = errors.New("httpclient: too many redirects")

// DenyPrivateIP is an AllowHost helper that rejects RFC1918, loopback,
// link-local and unique-local addresses — the standard SSRF guard.
// It does a single DNS resolution; rebinding attacks are still possible
// but require an active resolver-level exploit.
func DenyPrivateIP(host string) bool {
	// Strip an optional port.
	if i := strings.LastIndexByte(host, ':'); i >= 0 {
		// Avoid trimming the IPv6 colons by checking that what follows is numeric.
		port := host[i+1:]
		allDigit := port != ""
		for j := 0; j < len(port); j++ {
			if port[j] < '0' || port[j] > '9' {
				allDigit = false
				break
			}
		}
		if allDigit {
			host = host[:i]
		}
	}
	host = strings.TrimPrefix(strings.TrimSuffix(host, "]"), "[")

	ips, err := net.LookupIP(host)
	if err != nil || len(ips) == 0 {
		return false
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return false
		}
	}
	return true
}

func isPublicIP(ip net.IP) bool {
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() || ip.IsPrivate() || ip.IsUnspecified() {
		return false
	}
	// 169.254.169.254 (AWS/GCP metadata) is already caught by IsLinkLocalUnicast.
	return true
}

// DefaultRetryOn retries network errors and HTTP 5xx responses (except 501).
func DefaultRetryOn(resp *http.Response, err error) bool {
	if err != nil {
		// Don't retry context errors — the caller meant to bail out.
		var ue *url.Error
		if errors.As(err, &ue) {
			if errors.Is(ue.Err, context.Canceled) || errors.Is(ue.Err, context.DeadlineExceeded) {
				return false
			}
		}
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return false
		}
		return true
	}
	if resp == nil {
		return false
	}
	return resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented
}

// Client is a configured HTTP client with retry semantics.
type Client struct {
	*http.Client
	opts Options
	rng  *rand.Rand
}

// New returns a Client built from opts. Zero values fall back to defaults.
func New(opts Options) *Client {
	applyDefaults(&opts)

	transport := opts.Transport
	if transport == nil {
		transport = &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   opts.DialTimeout,
				KeepAlive: opts.KeepAlive,
			}).DialContext,
			MaxIdleConns:        opts.MaxIdleConns,
			MaxIdleConnsPerHost: opts.MaxIdleConnsPerHost,
			IdleConnTimeout:     opts.IdleConnTimeout,
			ForceAttemptHTTP2:   true,
		}
	}
	c := &Client{
		Client: &http.Client{
			Timeout:       opts.Timeout,
			Transport:     transport,
			CheckRedirect: buildCheckRedirect(opts),
		},
		opts: opts,
		rng:  rand.New(rand.NewSource(time.Now().UnixNano())),
	}
	return c
}

// buildCheckRedirect enforces MaxRedirects (-1 = disabled) and runs every
// redirect target through AllowHost when set.
func buildCheckRedirect(opts Options) func(req *http.Request, via []*http.Request) error {
	return func(req *http.Request, via []*http.Request) error {
		// Hop limit. The stdlib's default is 10; we mirror it when MaxRedirects==0.
		max := opts.MaxRedirects
		switch {
		case max < 0:
			return http.ErrUseLastResponse // disallow any redirect
		case max == 0:
			max = 10
		}
		if len(via) >= max {
			return ErrTooManyRedirects
		}
		if opts.AllowHost != nil && !opts.AllowHost(req.URL.Host) {
			return fmt.Errorf("%w: %s", ErrHostNotAllowed, req.URL.Host)
		}
		return nil
	}
}

func applyDefaults(o *Options) {
	if o.Timeout == 0 {
		o.Timeout = 30 * time.Second
	}
	if o.DialTimeout == 0 {
		o.DialTimeout = 5 * time.Second
	}
	if o.KeepAlive == 0 {
		o.KeepAlive = 30 * time.Second
	}
	if o.MaxIdleConns == 0 {
		o.MaxIdleConns = 100
	}
	if o.MaxIdleConnsPerHost == 0 {
		o.MaxIdleConnsPerHost = 32
	}
	if o.IdleConnTimeout == 0 {
		o.IdleConnTimeout = 90 * time.Second
	}
	if o.RetryWaitMin == 0 {
		o.RetryWaitMin = 100 * time.Millisecond
	}
	if o.RetryWaitMax == 0 {
		o.RetryWaitMax = 2 * time.Second
	}
	if o.RetryOn == nil {
		o.RetryOn = DefaultRetryOn
	}
}

// Do executes req with retry semantics. The request body must be re-readable
// across attempts; bodies created via bytes.NewReader / strings.NewReader are
// fine. For arbitrary io.Reader bodies, retries are disabled.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.opts.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.opts.UserAgent)
	}
	if c.opts.AllowHost != nil && !c.opts.AllowHost(req.URL.Host) {
		return nil, fmt.Errorf("%w: %s", ErrHostNotAllowed, req.URL.Host)
	}

	body, canRetry := snapshotBody(req)
	attempts := 1 + c.opts.MaxRetries
	if !canRetry {
		attempts = 1
	}

	var lastResp *http.Response
	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			if body != nil {
				req.Body = io.NopCloser(bytes.NewReader(body))
			}
			if err := c.sleep(req.Context(), attempt); err != nil {
				return nil, err
			}
		}
		lastResp, lastErr = c.Client.Do(req)
		if attempt+1 == attempts {
			break
		}
		if !c.opts.RetryOn(lastResp, lastErr) {
			break
		}
		if lastResp != nil {
			_ = lastResp.Body.Close()
			lastResp = nil
		}
	}
	return lastResp, lastErr
}

func snapshotBody(req *http.Request) ([]byte, bool) {
	if req.Body == nil || req.Body == http.NoBody {
		return nil, true
	}
	if req.GetBody != nil {
		// Use http's snapshotting machinery.
		return nil, true
	}
	b, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		return nil, false
	}
	req.Body = io.NopCloser(bytes.NewReader(b))
	return b, true
}

func (c *Client) sleep(ctx context.Context, attempt int) error {
	d := c.backoff(attempt)
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func (c *Client) backoff(attempt int) time.Duration {
	// Exponential with full jitter: random in [min, min*2^(attempt-1)], capped at max.
	min := c.opts.RetryWaitMin
	cap := c.opts.RetryWaitMax
	mult := time.Duration(1) << (attempt - 1)
	if mult <= 0 || min*mult > cap {
		return cap
	}
	upper := min * mult
	jitter := time.Duration(c.rng.Int63n(int64(upper-min) + 1))
	return min + jitter
}

// GetJSON issues GET url and decodes the body into out (JSON).
func (c *Client) GetJSON(ctx context.Context, url string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("httpclient: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	return c.doDecode(req, out)
}

// PostJSON marshals body to JSON, posts it to url, and decodes the response
// into out. Pass out=nil to discard the response body.
func (c *Client) PostJSON(ctx context.Context, url string, body, out any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("httpclient: marshal body: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("httpclient: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return c.doDecode(req, out)
}

func (c *Client) doDecode(req *http.Request, out any) error {
	resp, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("httpclient: %s %s: %w", req.Method, req.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return &HTTPError{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: resp.StatusCode,
			Body:       string(body),
		}
	}
	if out == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("httpclient: decode body: %w", err)
	}
	return nil
}

// HTTPError is returned by GetJSON / PostJSON when the response is >= 400.
type HTTPError struct {
	Method     string
	URL        string
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("httpclient: %s %s: %d %s", e.Method, e.URL, e.StatusCode, http.StatusText(e.StatusCode))
}
