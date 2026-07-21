package events

import "encoding/json"

// AvatarUploadedEvent defines avatar uploaded event.
type AvatarUploadedEvent struct {
	// AvatarID stores the avatar i d value.
	AvatarID string `json:"avatar_id"`
}

// Read decodes bytes into the receiver.
func (e *AvatarUploadedEvent) Read(data []byte) error {
	return json.Unmarshal(data, e)
}

// Bytes encodes the receiver as bytes.
func (e *AvatarUploadedEvent) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

// Version is the application version set at build time.
func (e *AvatarUploadedEvent) Version() []byte {
	return []byte("v1")
}

// String returns the string representation.
func (e *AvatarUploadedEvent) String() string {
	return "AvatarUploadedEvent"
}
