package events

type Eventer interface {
	Bytes() ([]byte, error)
	Version() []byte
}
