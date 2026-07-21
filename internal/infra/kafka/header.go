// Package kafka contains Kafka messaging helpers.
package kafka

import "github.com/twmb/franz-go/pkg/kgo"

// EventTypeHeaderKey defines the event type header key value.
const EventTypeHeaderKey = "X-Event-Type"

// EventIDHeaderKey defines the event i d header key value.
const EventIDHeaderKey = "X-Event-ID"

// VersionHeaderKey defines the version header key value.
const VersionHeaderKey = "X-Version"

// ContentTypeHeaderKey defines the content type header key value.
const ContentTypeHeaderKey = "Content-Type"

// Header represents a Kafka header key-value pair.
type Header struct {
	// Key stores the key value.
	Key string
	// Value stores the header value.
	Value string
}

// Headers is a collection of Kafka headers.
type Headers map[string]Header

// NewHeaders converts Kafka record headers into a map.
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

// Value returns a header value by key.
func (h Headers) Value(key string) (string, bool) {
	hv, ok := h[key]
	if !ok {
		return "", false
	}
	return hv.Value, true
}
