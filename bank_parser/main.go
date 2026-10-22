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
	CardID              *string
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
	ErrAmbiguous = errors.New("header is ambiguous")
)

var bom = [3]byte{239, 187, 191}

// Registers a guess for a specific bank sheet using some starting bytes
// Automatically adjusts for BOM bytes
func RegisterHeaderGuess(header string, id string, parser ParserFunc) {
	headerB := []byte(header)
	allGuesses = append(allGuesses, &guesser{
		size: len(headerB) + 3, // for BOM
		isRight: func(b []byte) bool {
			if slices.Equal(bom[:3], b[:3]) {
				b = b[3:]
			}

			return slices.Equal(b, headerB)
		},
		parser: parser,
		id:     id,
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

	switch len(goodParsers) {
	case 0:
		return nil, nil, nil
	case 1:
		return buf, goodParsers[0], nil
	default:
		return buf, nil, ErrAmbiguous
	}
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
