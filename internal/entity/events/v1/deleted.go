// Package events defines version 1 avatar event payloads.
package events

import "encoding/json"

// AvatarDeleted is the task type for deleted avatar events.
type AvatarDeleted struct {
	// UserID stores the user identifier.
	UserID string `json:"user_id"`
	// AvatarID stores the avatar i d value.
	AvatarID string `json:"avatar_id"`
	// S3Key stores the s3 key value.
	S3Key []string `json:"s3_keys"`
}

// Read decodes bytes into the receiver.
func (e *AvatarDeleted) Read(data []byte) error {
	return json.Unmarshal(data, e)
}

// Bytes encodes the receiver as bytes.
func (e *AvatarDeleted) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

// Version is the application version set at build time.
func (e *AvatarDeleted) Version() []byte {
	return []byte("v1")
}

// String returns the string representation.
func (e *AvatarDeleted) String() string {
	return "AvatarDeleted"
}
