package events

import "encoding/json"

type AvatarUploadedEvent struct {
	AvatarID string `json:"avatar_id"`
}

func (e *AvatarUploadedEvent) Read(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *AvatarUploadedEvent) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

func (e *AvatarUploadedEvent) Version() []byte {
	return []byte("v1")
}

func (e *AvatarUploadedEvent) String() string {
	return "AvatarUploadedEvent"
}
