package bank_parser

import (
	"context"
	"errors"
	"io"
	"iter"
	"slices"
	"time"
)

type Transaction struct {
	AuthedAt, SettledAt time.Time
	Description         string
	Amt                 float64
	AmtAfterTransaction *float64
}

type ParserFunc func(ctx context.Context, r io.Reader) (iter.Seq[*Transaction], error)

type guesser struct {
	size    int
	isRight func([]byte) bool
	id      string
	parser  ParserFunc
}

var allGuesses = []*guesser{}

var (
	ErrAmbiguous = errors.New("Header is ambiguous")
)

func RegisterHeaderGuess(header string, id string, parser ParserFunc) {
	headerB := []byte(header)
	allGuesses = append(allGuesses, &guesser{
		size:    len(header),
		isRight: func(b []byte) bool { return slices.Equal(b, headerB) },
		parser:  parser,
	})

	slices.SortFunc(allGuesses, func(a, b *guesser) int {
		return b.size - a.size
	})
}

type bytesFirst struct {
	buf []byte
	r   io.Reader
}

func (r *bytesFirst) Read(p []byte) (n int, err error) {
	if r.buf != nil {
		n := copy(p, r.buf)
		if len(r.buf) == n {
			r.buf = nil
		} else {
			r.buf = r.buf[n:]
		}

		return n, nil
	}

	return r.r.Read(p)
}

func guess(r io.Reader) ([]byte, *guesser, error) {
	if len(allGuesses) == 0 {
		return nil, nil, nil
	}

	buf := make([]byte, allGuesses[0].size)
	_, err := r.Read(buf)
	if err != nil {
		return nil, nil, err
	}

	goodParsers := []*guesser{}
	for _, v := range allGuesses {
		if v.isRight(buf[:v.size]) {
			goodParsers = append(goodParsers, v)
		}
	}

	if len(goodParsers) != 1 {
		println("Meow", len(goodParsers), string(buf))
		return buf, nil, ErrAmbiguous
	}

	return buf, goodParsers[0], nil
}

// Get a guesser id from the reader. This is unlikely to be useful, as it reads directly from r
// This may be useful for testing/debug. See [Iter] for actual usage
func GuessID(r io.Reader) (string, error) {
	_, g, err := guess(r)
	if err != nil || g == nil {
		return "", err
	}

	return g.id, nil
}

func Iter(ctx context.Context, r io.Reader) (iter.Seq[*Transaction], error) {
	buf, g, err := guess(r)
	if err != nil || g == nil {
		return nil, err
	}

	return g.parser(ctx, &bytesFirst{buf, r})
}
