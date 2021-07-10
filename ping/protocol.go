package ping

type Protocol int

const (
	TCP Protocol = iota
)

func (p Protocol) String() string {
	switch p {
	case TCP:
		return "tcp"
	}
	return "unknown"
}
