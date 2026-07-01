package kafka

import "github.com/twmb/franz-go/pkg/kgo"

const EventTypeHeaderKey = "X-Event-Type"

type Header struct {
	Key   string
	Value string
}
type Headers map[string]Header

func NewHeaders(headers ...kgo.RecordHeader) Headers {
	m := make(map[string]Header, len(headers))
	for _, h := range headers {
		m[h.Key] = Header{
			Key:   h.Key,
			Value: string(h.Value),
		}
	}
	return m
}

func (h Headers) Value(key string) (string, bool) {
	hv, ok := h[key]
	if !ok {
		return "", false
	}
	return hv.Value, true
}
