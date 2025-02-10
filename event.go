package pulse

type Event interface {
	Kind() string
}
