package events

import "encoding/json"

type AvatarDeleted struct {
	UserID   string   `json:"user_id"`
	AvatarID string   `json:"avatar_id"`
	S3Key    []string `json:"s3_keys"`
}

func (e *AvatarDeleted) Read(data []byte) error {
	return json.Unmarshal(data, e)
}

func (e *AvatarDeleted) Bytes() ([]byte, error) {
	return json.Marshal(e)
}

func (e *AvatarDeleted) Version() []byte {
	return []byte("v1")
}

func (e *AvatarDeleted) String() string {
	return "AvatarDeleted"
}
