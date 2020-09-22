package ebuf_test

import (
	"bytes"
	"testing"
	"time"

	"github.com/negli0/ebuf"
)

func TestDatagramBufReadWrite(t *testing.T) {
	tests := []struct {
		input    []byte
		size     int
		expected []byte
	}{
		{
			// the receiving size is the same as the datagram size
			[]byte("hello"),
			5,
			[]byte("hello"),
		},
		{
			// the receiving size is less than the datagram size
			[]byte("hello"),
			1,
			[]byte("h"),
		},
		{
			// the receiving size is larger than the datagram size
			[]byte("hello"),
			10,
			[]byte("hello\x00\x00\x00\x00\x00"),
		},
	}

	dbuf := ebuf.NewDatagramBuf(1)

	go func() {
		for i, test := range tests {
			// バッファに書き込み
			n, err := dbuf.Write(test.input)
			if err != nil {
				t.Errorf("[error] [Datagram Buffer] [Write %d]: %v", i, err)
			}
			t.Logf("[Datagram Bufffer] [Write %d]: %s (%d byte)\n", i, test.input, n)
		}
	}()

	for i, test := range tests {
		actual := make([]byte, test.size)
		n, err := dbuf.Read(actual)
		if err != nil {
			t.Errorf("[error] [Datagram Buffer] [Read %d]: %v", i, err)
		}
		t.Logf("[Datagram Bufffer] [Read %d]: %s (%d byte)\n", i, actual, n)
		if !bytes.Equal(test.expected, actual) {
			t.Errorf("expected %v (got %v)", test.expected, actual)
		}
	}
}

func TestStreamBufReadWrite(t *testing.T) {
	type result struct {
		size  int
		value []byte
	}
	tests := []struct {
		inputs   [][]byte
		expected []result
	}{
		// 要素: chunk, nrChunks, size
		// 1 chunk (size: 6) を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("abcdef")},
			[]result{
				{1, []byte("a")}, {2, []byte("bc")}, {3, []byte("def")},
			},
		},

		// 2 chunk (size: 3, 3) を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("abc"), []byte("def")},
			[]result{
				{3, []byte("abc")}, {3, []byte("def")},
			},
		},
		// 3 chunk (size: 1, 2, 3)を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("a"), []byte("bc"), []byte("def")},
			[]result{
				{1, []byte("a")}, {2, []byte("bc")}, {3, []byte("def")},
			},
		},
		// 3 chunk (size: 1, 3, 2)を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("a"), []byte("bcd"), []byte("ef")},
			[]result{
				{1, []byte("a")}, {2, []byte("bc")}, {3, []byte("def")},
			},
		},
		// 3 chunk (size: 2, 3, 5)を 1, 5, 4 バイトずつ読む
		{
			[][]byte{[]byte("ab"), []byte("cde"), []byte("fghij")},
			[]result{
				{1, []byte("a")}, {5, []byte("bcdef")}, {4, []byte("ghij")},
			},
		},
		// 5 chunk (size: 2, 3, 3, 2, 4)を 1, 10, 3 バイトずつ読む
		{
			[][]byte{[]byte("ab"), []byte("cde"), []byte("fgh"), []byte("ij"), []byte("klmn")},
			[]result{
				{1, []byte("a")}, {10, []byte("bcdefghijk")}, {3, []byte("lmn")},
			},
		},
		// 1 chunk (size: 2) を 5 バイト読む
		{
			[][]byte{[]byte("ab")},
			[]result{
				{5, []byte("ab\x00\x00\x00")},
			},
		},
	}

	for i, test := range tests {
		sbuf := ebuf.NewStreamBuf(5)
		start := make(chan struct{})
		done := make(chan struct{})
		go func(i int, sbuf *ebuf.StreamBuf, start, done chan struct{}) {
			// バッファに書き込み
			for j, in := range test.inputs {
				n, err := sbuf.Write(in)
				if err != nil {
					t.Errorf("[error] [Stream Buffer] [Write %d-%d]: %v", i, j, err)
				}
				t.Logf("[Stream Bufffer] [Write %d-%d]: %s (%d byte)\n", i, j, in, n)
			}
			close(start)
			<-done
		}(i, sbuf, start, done)

		<-start
		for j, ex := range test.expected {
			actual := make([]byte, ex.size)
			n, err := sbuf.Read(actual)
			if err != nil {
				t.Errorf("[error] [Stream Buffer] [Read %d-%d]: %v", i, j, err)
			}
			t.Logf("[Stream Bufffer] [Read %d-%d]: %s (%d byte)\n", i, j, actual, n)
			if !bytes.Equal(ex.value, actual) {
				t.Errorf("expected %v (got %v)", ex.value, actual)
			}
		}
		close(done)
	}

}

func TestStreamBufBlockingRead(t *testing.T) {
	type result struct {
		size  int
		value []byte
	}
	tests := []struct {
		inputs   [][]byte
		expected []result
	}{
		// 要素: chunk, nrChunks, size
		// 1 chunk (size: 6) を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("abcdef")},
			[]result{
				{1, []byte("a")}, {2, []byte("bc")}, {3, []byte("def")},
			},
		},

		// 2 chunk (size: 3, 3) を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("abc"), []byte("def")},
			[]result{
				{3, []byte("abc")}, {3, []byte("def")},
			},
		},
		// 3 chunk (size: 1, 2, 3)を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("a"), []byte("bc"), []byte("def")},
			[]result{
				{1, []byte("a")}, {2, []byte("bc")}, {3, []byte("def")},
			},
		},
		// 3 chunk (size: 1, 3, 2)を 1, 2, 3 バイトずつ読む
		{
			[][]byte{[]byte("a"), []byte("bcd"), []byte("ef")},
			[]result{
				{1, []byte("a")}, {2, []byte("bc")}, {3, []byte("def")},
			},
		},
		// 3 chunk (size: 2, 3, 5)を 1, 5, 4 バイトずつ読む
		{
			[][]byte{[]byte("ab"), []byte("cde"), []byte("fghij")},
			[]result{
				{1, []byte("a")}, {5, []byte("bcdef")}, {4, []byte("ghij")},
			},
		},
		// 5 chunk (size: 2, 3, 3, 2, 4)を 1, 10, 3 バイトずつ読む
		{
			[][]byte{[]byte("ab"), []byte("cde"), []byte("fgh"), []byte("ij"), []byte("klmn")},
			[]result{
				{1, []byte("a")}, {10, []byte("bcdefghijk")}, {3, []byte("lmn")},
			},
		},
		// 1 chunk (size: 2) を 5 バイト読む
		{
			[][]byte{[]byte("ab")},
			[]result{
				{5, []byte("ab")},
			},
		},
	}

	for i, test := range tests {
		sbuf := ebuf.NewStreamBuf(5)
		go func(i int, sbuf *ebuf.StreamBuf) {
			// バッファに書き込み
			for j, in := range test.inputs {
				time.Sleep(time.Millisecond)
				n, err := sbuf.Write(in)
				if err != nil {
					t.Errorf("[error] [Stream Buffer] [Write %d-%d]: %v", i, j, err)
				}
				t.Logf("[Stream Bufffer] [Write %d-%d]: %s (%d byte)\n", i, j, in, n)
			}
		}(i, sbuf)

		for j, ex := range test.expected {
			actual := make([]byte, ex.size)
			var total int
			for total != len(ex.value) {
				n, err := sbuf.Read(actual[total:])
				if err != nil {
					t.Errorf("[error] [Stream Buffer] [Read %d-%d]: %v", i, j, err)
				}
				total += n
			}
			t.Logf("[Stream Bufffer] [Read %d-%d]: %s (%d byte)\n", i, j, actual[:total], total)
			if !bytes.Equal(ex.value, actual[:total]) {
				t.Errorf("expected %v (got %v)", ex.value, actual[:total])
			}
		}
	}
}
