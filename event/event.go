package event

type Reader interface {
	Read() <-chan []byte
}

type Writer interface {
	Write(b []byte) (int, error)
}

type Closer interface {
	Close() error
}

type ReadWriteCloser interface {
	Reader
	Writer
	Closer
}
