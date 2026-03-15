package kafka

import (
	"context"

	"github.com/twmb/franz-go/pkg/kgo"
	"go.opentelemetry.io/otel"
)

type kafkaCarrier struct {
	record *kgo.Record
}

func (c *kafkaCarrier) Get(key string) string {
	for _, h := range c.record.Headers {
		if h.Key == key {
			return string(h.Value)
		}
	}
	return ""
}

func (c *kafkaCarrier) Set(key, value string) {
	for i, h := range c.record.Headers {
		if h.Key == key {
			c.record.Headers[i].Value = []byte(value)
			return
		}
	}
	c.record.Headers = append(c.record.Headers, kgo.RecordHeader{
		Key:   key,
		Value: []byte(value),
	})
}

func (c *kafkaCarrier) Keys() []string {
	keys := make([]string, len(c.record.Headers))
	for i, h := range c.record.Headers {
		keys[i] = h.Key
	}
	return keys
}

func InjectTraceContext(ctx context.Context, record *kgo.Record) {
	otel.GetTextMapPropagator().Inject(ctx, &kafkaCarrier{record: record})
}

func ExtractTraceContext(ctx context.Context, record *kgo.Record) context.Context {
	return otel.GetTextMapPropagator().Extract(ctx, &kafkaCarrier{record: record})
}
