// Package ebuf provides some enhanced buffer structures, such as
// channel-based datagram buffer, channel-based byte-stream buffer.
package ebuf

import "errors"

type chbuf chan []byte

var (
	// ErrBrokenBuffer shows the buffer is broken.
	ErrBrokenBuffer = errors.New("buffer is broken")
)

// DatagramBuf is channel-based datagram buffer.
type DatagramBuf struct {
	chbuf
}

// StreamBuf is channel-based byte-stream buffer.
type StreamBuf struct {
	chbuf
	rest []byte
}

// NewDatagramBuf generates a new DatagramBuf which can buffer `nrDgrams` datagrams.
func NewDatagramBuf(nrDgrams int) *DatagramBuf {
	var dbuf DatagramBuf
	dbuf.chbuf = make(chan []byte, nrDgrams)
	return &dbuf
}

// Write implements io.Writer. Write will be blocked when
// the inner channel is full.
func (b *DatagramBuf) Write(p []byte) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			n, err = 0, ErrBrokenBuffer
			return
		}
	}()

	cp := make([]byte, len(p))
	copy(cp, p)
	n, err = len(cp), nil
	b.chbuf <- cp

	return n, err
}

// Read implements io.Reader. Read reads one
// datagram from its inner channel, and stores it to p.
// If len(p) is smaller than the received datagram,
// Read copies the largest possible size of data.
// In this case, the rest of the datagram is discarded.
// If len(p) is larger than the received datagram,
// the unused field of p is left. Therefore, the caller
// must treat `n` as the size of received datagram.
// Read will be blocked when the inner channel is empty.
func (b *DatagramBuf) Read(p []byte) (n int, err error) {
	r, ok := <-b.chbuf
	if !ok {
		return 0, ErrBrokenBuffer
	}
	return copy(p, r), nil
}

// NewStreamBuf generates a new StreamBuf which can buffer `nrChunks` chunks.
// StreamBuf provides the byte-stream with the caller by concatenating a seriese of chunks.
func NewStreamBuf(nrChunks int) *StreamBuf {
	var sb StreamBuf
	sb.chbuf = make(chan []byte, nrChunks)
	sb.rest = []byte{}
	return &sb
}

// Read implements io.Reader. Read reads len(p) bytes from StreamBuf.
// If len(p) is larger than the length of buffered data, Read
// reads the all buffered data and returns the length of data in byte.
// Therefore, Read will not be blocked. When needed to read
// a specified length, it is better to use io.ReadAtLeast() together.
func (b *StreamBuf) Read(p []byte) (int, error) {
	requiredLen := len(p)
	provideLen := requiredLen

	if len(b.rest) >= requiredLen {
		// this StreamBuf can return the required length of bytes without fetching from its inner channel
		copy(p, b.rest[:provideLen])
		b.rest = b.rest[provideLen:]

		return provideLen, nil
	}

	// StreamBuf tries fetching more bytes from its inner channel
	// until the the length of the rest slice is larger than the required length.
	// If no more bytes in the channel, StreamBuf returns the largest possible
	// length of data.
L:
	for len(b.rest) < requiredLen {
		select {
		case r, ok := <-b.chbuf:
			if !ok {
				return 0, ErrBrokenBuffer
			}
			b.rest = append(b.rest, r...)
		default:
			provideLen = len(b.rest)
			break L
		}
	}

	// If inner buffer is empty and the provideLen is zero,
	// Read will be blocked until StreamBuf fetches one chunk.
	if provideLen == 0 {
		r, ok := <-b.chbuf
		if !ok {
			return 0, ErrBrokenBuffer
		}
		b.rest = append(b.rest, r...)

		if len(b.rest) < requiredLen {
			provideLen = len(b.rest)
		} else {
			provideLen = requiredLen
		}
	}

	copy(p, b.rest[:provideLen])
	b.rest = b.rest[provideLen:]

	return provideLen, nil
}

// Write implements io.Writer. Write writes len(p) bytes to StreamBuf.
// When the StreamBuf is full, Write will be blocked.
func (b *StreamBuf) Write(p []byte) (n int, err error) {
	defer func() {
		if r := recover(); r != nil {
			n, err = 0, ErrBrokenBuffer
			return
		}
	}()

	cp := make([]byte, len(p))
	copy(cp, p)
	n, err = len(cp), nil
	b.chbuf <- cp

	return n, err
}
