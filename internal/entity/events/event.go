// Package events defines shared domain event contracts.
package events

// Eventer describes a serializable domain event.
type Eventer interface {
	// Bytes encodes the receiver as bytes.
	Bytes() ([]byte, error)
	// Version is the application version set at build time.
	Version() []byte
}
