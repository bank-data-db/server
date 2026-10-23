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
type internalParserFunc func(ctx context.Context, guesserCtx any, r io.Reader) (iter.Seq[*Transaction], error)

type guesser struct {
	size int
	// returns a guesser context & if this is right
	isRight func([]byte) (any, bool)
	id      string
	parser  internalParserFunc
}

var allGuesses = []*guesser{}

var (
	ErrAmbiguous = errors.New("header is ambiguous")
)

var bom = [3]byte{239, 187, 191}

type headerGuesserBOM bool

// Registers a guess for a specific bank sheet using some starting bytes
// Automatically adjusts for BOM bytes
func RegisterHeaderGuess(header string, id string, parser ParserFunc) {
	headerB := []byte(header)
	allGuesses = append(allGuesses, &guesser{
		size: len(headerB) + 3, // for BOM
		isRight: func(b []byte) (any, bool) {
			hasBOM := slices.Equal(bom[:3], b[:3])
			if hasBOM {
				return headerGuesserBOM(true), slices.Equal(b[3:], headerB)
			}
			return headerGuesserBOM(false), slices.Equal(b[:len(b)-3], headerB)
		},
		parser: func(ctx context.Context, guesserCtx any, r io.Reader) (iter.Seq[*Transaction], error) {
			if guesserCtx.(headerGuesserBOM) {
				var read = [3]byte{}
				_, err := r.Read(read[:])
				if err != nil {
					return nil, err
				}
			}

			return parser(ctx, r)
		},
		id: id,
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

func guess(r io.Reader) ([]byte, *guesser, any, error) {
	if len(allGuesses) == 0 {
		return nil, nil, nil, nil
	}

	buf := make([]byte, allGuesses[0].size)
	_, err := r.Read(buf)
	if err != nil {
		return nil, nil, nil, err
	}

	var guesserCtx any
	var goodGuesser *guesser
	for _, v := range allGuesses {
		if ctx, ok := v.isRight(buf[:v.size]); ok {
			if goodGuesser != nil {
				return buf, nil, nil, ErrAmbiguous
			}

			guesserCtx = ctx
			goodGuesser = v
		}
	}

	if goodGuesser == nil {
		return nil, nil, nil, nil
	}

	return buf, goodGuesser, guesserCtx, nil
}

// Get a guesser id from the reader. This is unlikely to be useful, as it reads directly from r
// This may be useful for testing/debug. See [Iter] for actual usage
func GuessID(r io.Reader) (string, error) {
	_, g, _, err := guess(r)
	if err != nil || g == nil {
		return "", err
	}

	return g.id, nil
}

func Iter(ctx context.Context, r io.Reader) (iter.Seq[*Transaction], error) {
	buf, g, gCtx, err := guess(r)
	if err != nil || g == nil {
		return nil, err
	}

	return g.parser(ctx, gCtx, &bytesFirst{buf, r})
}
