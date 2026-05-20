// Package elastic is a thin, generic wrapper around the official
// go-elasticsearch v8 client. It exposes Index / Get / Update / Delete /
// Search / Bulk / Scroll with ctx-first APIs.
package elastic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/esapi"
	"github.com/elastic/go-elasticsearch/v8/esutil"
)

// Client wraps an *elasticsearch.Client and exposes a small set of
// document-centric helpers.
type Client struct {
	raw *elasticsearch.Client
}

// Config mirrors the subset of elasticsearch.Config we expose.
type Config struct {
	Addresses []string
	Username  string
	Password  string
	APIKey    string
	CACert    []byte
}

// NewClient builds a new Client from the given config.
func NewClient(cfg Config) (*Client, error) {
	es, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: cfg.Addresses,
		Username:  cfg.Username,
		Password:  cfg.Password,
		APIKey:    cfg.APIKey,
		CACert:    cfg.CACert,
	})
	if err != nil {
		return nil, fmt.Errorf("elastic: new client: %w", err)
	}
	return &Client{raw: es}, nil
}

// Raw returns the underlying client so callers can use any feature not exposed here.
func (c *Client) Raw() *elasticsearch.Client { return c.raw }

// Ping checks the cluster is reachable.
func (c *Client) Ping(ctx context.Context) error {
	resp, err := c.raw.Ping(c.raw.Ping.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("elastic: ping: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

// Refresh policy values accepted by Index / IndexWithRefresh.
const (
	RefreshFalse   = "false"    // do not wait — fastest, default
	RefreshTrue    = "true"     // force a refresh before returning
	RefreshWaitFor = "wait_for" // block until the next refresh completes
)

// Index indexes a document with refresh=false (fire-and-forget).
// If id is empty, ES assigns one. For controlled visibility timing use
// IndexWithRefresh.
func (c *Client) Index(ctx context.Context, index, id string, body any) error {
	return c.IndexWithRefresh(ctx, index, id, body, RefreshFalse)
}

// IndexWithRefresh is Index with an explicit refresh policy.
// refresh must be one of: "false", "true", "wait_for".
func (c *Client) IndexWithRefresh(ctx context.Context, index, id string, body any, refresh string) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("elastic: marshal: %w", err)
	}
	req := esapi.IndexRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(data),
		Refresh:    refresh,
	}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("elastic: index: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

// Get fetches a document by id and decodes its _source into out.
// Returns (false, nil) when the document does not exist.
func (c *Client) Get(ctx context.Context, index, id string, out any) (bool, error) {
	req := esapi.GetRequest{Index: index, DocumentID: id}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return false, fmt.Errorf("elastic: get: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return false, nil
	}
	if err := checkResponse(resp); err != nil {
		return false, err
	}
	var envelope struct {
		Source json.RawMessage `json:"_source"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return true, fmt.Errorf("elastic: decode: %w", err)
	}
	if out != nil {
		if err := json.Unmarshal(envelope.Source, out); err != nil {
			return true, fmt.Errorf("elastic: unmarshal source: %w", err)
		}
	}
	return true, nil
}

// Update applies a partial document update.
func (c *Client) Update(ctx context.Context, index, id string, partial any) error {
	data, err := json.Marshal(map[string]any{"doc": partial})
	if err != nil {
		return fmt.Errorf("elastic: marshal: %w", err)
	}
	req := esapi.UpdateRequest{
		Index:      index,
		DocumentID: id,
		Body:       bytes.NewReader(data),
	}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("elastic: update: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

// Delete removes a document by id. A 404 is treated as success.
func (c *Client) Delete(ctx context.Context, index, id string) error {
	req := esapi.DeleteRequest{Index: index, DocumentID: id}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("elastic: delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil
	}
	return checkResponse(resp)
}

// SearchHit is one hit from a search response.
type SearchHit struct {
	ID     string          `json:"_id"`
	Index  string          `json:"_index"`
	Score  float64         `json:"_score"`
	Source json.RawMessage `json:"_source"`
}

// SearchResult holds the bits of a search response most callers care about.
type SearchResult struct {
	TotalHits int64       `json:"-"`
	Hits      []SearchHit `json:"-"`

	// Raw response for callers needing aggregations or other fields.
	Raw json.RawMessage `json:"-"`
}

// Search runs a raw query (a JSON-serializable structure matching the ES
// search request body) against one or more indices.
func (c *Client) Search(ctx context.Context, indices []string, query any) (*SearchResult, error) {
	data, err := json.Marshal(query)
	if err != nil {
		return nil, fmt.Errorf("elastic: marshal query: %w", err)
	}
	resp, err := c.raw.Search(
		c.raw.Search.WithContext(ctx),
		c.raw.Search.WithIndex(indices...),
		c.raw.Search.WithBody(bytes.NewReader(data)),
	)
	if err != nil {
		return nil, fmt.Errorf("elastic: search: %w", err)
	}
	defer resp.Body.Close()
	if err := checkResponse(resp); err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("elastic: read body: %w", err)
	}
	var envelope struct {
		Hits struct {
			Total struct {
				Value int64 `json:"value"`
			} `json:"total"`
			Hits []SearchHit `json:"hits"`
		} `json:"hits"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("elastic: decode search: %w", err)
	}
	return &SearchResult{
		TotalHits: envelope.Hits.Total.Value,
		Hits:      envelope.Hits.Hits,
		Raw:       raw,
	}, nil
}

// BulkItem is one document to send through the BulkIndexer.
type BulkItem struct {
	Action string // "index", "create", "update", "delete"
	Index  string
	ID     string
	Body   any
}

// Bulk runs a batch of operations. It uses the official BulkIndexer for
// efficient batching and backpressure.
func (c *Client) Bulk(ctx context.Context, items []BulkItem) (stats esutil.BulkIndexerStats, err error) {
	if len(items) == 0 {
		return esutil.BulkIndexerStats{}, nil
	}
	bi, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Client: c.raw,
	})
	if err != nil {
		return stats, fmt.Errorf("elastic: new bulk indexer: %w", err)
	}

	for _, it := range items {
		var body io.ReadSeeker
		if it.Body != nil {
			data, err := json.Marshal(it.Body)
			if err != nil {
				return stats, fmt.Errorf("elastic: marshal bulk item %s: %w", it.ID, err)
			}
			body = bytes.NewReader(data)
		}
		if err := bi.Add(ctx, esutil.BulkIndexerItem{
			Action:     it.Action,
			Index:      it.Index,
			DocumentID: it.ID,
			Body:       body,
		}); err != nil {
			return stats, fmt.Errorf("elastic: bulk add: %w", err)
		}
	}
	if err := bi.Close(ctx); err != nil {
		return stats, fmt.Errorf("elastic: bulk close: %w", err)
	}
	return bi.Stats(), nil
}

// ScrollFunc is invoked for each hit during Scroll. Return an error to stop.
type ScrollFunc func(hit SearchHit) error

// Scroll iterates every document matching query, calling fn for each hit.
// pageSize controls how many hits are pulled per scroll page.
func (c *Client) Scroll(ctx context.Context, indices []string, query any, pageSize int, fn ScrollFunc) error {
	data, err := json.Marshal(query)
	if err != nil {
		return fmt.Errorf("elastic: marshal query: %w", err)
	}
	if pageSize <= 0 {
		pageSize = 500
	}

	resp, err := c.raw.Search(
		c.raw.Search.WithContext(ctx),
		c.raw.Search.WithIndex(indices...),
		c.raw.Search.WithBody(bytes.NewReader(data)),
		c.raw.Search.WithSize(pageSize),
		c.raw.Search.WithScroll(scrollKeepalive),
	)
	if err != nil {
		return fmt.Errorf("elastic: scroll search: %w", err)
	}
	if err := checkResponse(resp); err != nil {
		resp.Body.Close()
		return err
	}

	var page scrollPage
	if err := json.NewDecoder(resp.Body).Decode(&page); err != nil {
		resp.Body.Close()
		return fmt.Errorf("elastic: decode scroll page: %w", err)
	}
	resp.Body.Close()

	scrollID := page.ScrollID
	defer func() {
		if scrollID != "" {
			_, _ = c.raw.ClearScroll(c.raw.ClearScroll.WithScrollID(scrollID))
		}
	}()

	for {
		for _, h := range page.Hits.Hits {
			if err := fn(h); err != nil {
				return err
			}
		}
		if len(page.Hits.Hits) == 0 {
			return nil
		}

		next, err := c.raw.Scroll(
			c.raw.Scroll.WithContext(ctx),
			c.raw.Scroll.WithScrollID(scrollID),
			c.raw.Scroll.WithScroll(scrollKeepalive),
		)
		if err != nil {
			return fmt.Errorf("elastic: scroll: %w", err)
		}
		if err := checkResponse(next); err != nil {
			next.Body.Close()
			return err
		}
		page = scrollPage{}
		if err := json.NewDecoder(next.Body).Decode(&page); err != nil {
			next.Body.Close()
			return fmt.Errorf("elastic: decode scroll: %w", err)
		}
		next.Body.Close()
		scrollID = page.ScrollID
	}
}

const scrollKeepalive = time.Minute

type scrollPage struct {
	ScrollID string `json:"_scroll_id"`
	Hits     struct {
		Hits []SearchHit `json:"hits"`
	} `json:"hits"`
}

// CreateIndex creates an index with the given mapping/settings body.
// If the index already exists, the call is a no-op.
func (c *Client) CreateIndex(ctx context.Context, name string, body any) error {
	exists, err := c.IndexExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("elastic: marshal mapping: %w", err)
	}
	req := esapi.IndicesCreateRequest{
		Index: name,
		Body:  bytes.NewReader(data),
	}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("elastic: create index: %w", err)
	}
	defer resp.Body.Close()
	return checkResponse(resp)
}

// DeleteIndex deletes an index. A 404 is treated as success.
func (c *Client) DeleteIndex(ctx context.Context, name string) error {
	req := esapi.IndicesDeleteRequest{Index: []string{name}}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return fmt.Errorf("elastic: delete index: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == 404 {
		return nil
	}
	return checkResponse(resp)
}

// IndexExists reports whether the named index exists.
func (c *Client) IndexExists(ctx context.Context, name string) (bool, error) {
	req := esapi.IndicesExistsRequest{Index: []string{name}}
	resp, err := req.Do(ctx, c.raw)
	if err != nil {
		return false, fmt.Errorf("elastic: exists: %w", err)
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case 200:
		return true, nil
	case 404:
		return false, nil
	default:
		return false, errFromResponse(resp)
	}
}

func checkResponse(resp *esapi.Response) error {
	if !resp.IsError() {
		return nil
	}
	return errFromResponse(resp)
}

func errFromResponse(resp *esapi.Response) error {
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("elastic: %s: %s", resp.Status(), strings.TrimSpace(string(body)))
}

// ErrNotFound is returned (wrapped) for some not-found conditions where the
// caller asked for an explicit error. Most helpers return (false, nil) on 404.
var ErrNotFound = errors.New("not found")
