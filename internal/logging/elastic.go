package logging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

type ElasticHandler struct {
	client *elasticsearch.Client
	index  string
	level  slog.Level
	attrs  []slog.Attr
	group  string
}

func NewElasticHandler(addresses []string, index string) (*ElasticHandler, error) {
	client, err := elasticsearch.NewClient(elasticsearch.Config{
		Addresses: addresses,
	})
	if err != nil {
		return nil, fmt.Errorf("elasticsearch client: %w", err)
	}

	return &ElasticHandler{
		client: client,
		index:  index,
		level:  slog.LevelInfo,
	}, nil
}

func (h *ElasticHandler) Enabled(_ context.Context, level slog.Level) bool {
	return level >= h.level
}

func (h *ElasticHandler) Handle(_ context.Context, r slog.Record) error {
	doc := map[string]any{
		"timestamp": r.Time.Format(time.RFC3339Nano),
		"level":     r.Level.String(),
		"message":   r.Message,
		"service":   "payment-service",
	}

	if h.group != "" {
		group := make(map[string]any)
		r.Attrs(func(a slog.Attr) bool {
			group[a.Key] = a.Value.Any()
			return true
		})
		for _, a := range h.attrs {
			group[a.Key] = a.Value.Any()
		}
		doc[h.group] = group
	} else {
		r.Attrs(func(a slog.Attr) bool {
			doc[a.Key] = a.Value.Any()
			return true
		})
		for _, a := range h.attrs {
			doc[a.Key] = a.Value.Any()
		}
	}

	body, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	res, err := h.client.Index(h.index, bytes.NewReader(body))
	if err != nil {
		return err
	}
	defer res.Body.Close()
	return nil
}

func (h *ElasticHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ElasticHandler{
		client: h.client,
		index:  h.index,
		level:  h.level,
		attrs:  append(h.attrs, attrs...),
		group:  h.group,
	}
}

func (h *ElasticHandler) WithGroup(name string) slog.Handler {
	return &ElasticHandler{
		client: h.client,
		index:  h.index,
		level:  h.level,
		attrs:  h.attrs,
		group:  name,
	}
}
