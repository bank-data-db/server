// https://caixadirectaonline.cgd.pt
package cgd

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"strconv"
	"strings"
	"time"

	"github.com/shadiestgoat/bankDataDB/bank_parser"
	"github.com/shadiestgoat/bankDataDB/log"
)

func parseNum(l log.Logger, portuguese bool, s string) (float64, error) {
	if s == "" {
		return 0, nil
	}

	if portuguese {
		s = strings.ReplaceAll(s, ".", "")
		s = strings.ReplaceAll(s, ",", ".")
	} else {
		s = strings.ReplaceAll(s, ",", "")
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		l.Warnf("Can't parse number (%s): %v", s, err)
		return f, err
	}

	return f, nil
}

func skipSection(r io.Reader) error {
	newlines := 0

	for {
		buf := [1]byte{}
		// it is assumed that the reader is buffered
		n, err := r.Read(buf[:])
		if errors.Is(err, io.EOF) || n != 1 {
			return nil
		} else if err != nil {
			return err
		}

		switch buf[0] {
		case '\r':
			continue
		case '\n':
			newlines++
			if newlines == 2 {
				return nil
			}
		default:
			newlines = 0
		}
	}
}

func init() {
	bank_parser.RegisterHeaderGuess("Consultar saldos e movimentos", "cgd/pt", NewParser)
	bank_parser.RegisterHeaderGuess("View current operations and balances", "cgd/en", NewParser)
}

func NewParser(ctx context.Context, r io.Reader, slog log.Logger) (iter.Seq[*bank_parser.Transaction], error) {
	err := skipSection(r)
	if err != nil {
		return nil, err
	}

	v := make([]byte, 9)
	r.Read(v)
	portuguese := false
	off := 7

	if bytes.HasPrefix(v, []byte("Conta")) {
		portuguese = true
		off = 5
	}

	if v[off] == ' ' {
		off++
	}

	split := string([]byte{v[off]})

	err = skipSection(r)
	if err != nil {
		return nil, err
	}

	sc := bufio.NewScanner(r)
	sc.Scan() // skip header

	return func(yield func(*bank_parser.Transaction) bool) {
		for sc.Scan() {
			l := sc.Text()
			if l == "" || strings.HasPrefix(l, "\t") || strings.HasPrefix(l, " ") {
				break
			}
			if l[len(l)-1] == '\r' {
				l = l[:len(l)-1]
			}

			// Op. Date 	Value Date 	Description 	Debit 	Credit 	Balance Accounting 	Balance available 	Categoria (EN)
			cols := strings.Split(l, split)
			if len(cols) < 8 {
				slog.Warnf("Wrong # of columns (expected 8, got %d)", len(cols))
				continue
			}

			authedAt, err := time.Parse("02-01-2006", cols[1])
			if err != nil {
				slog.Warnf("Can't parse date (%v): %v", cols[1], err)
				continue
			}

			settledAt, err := time.Parse("02-01-2006", cols[0])
			if err != nil {
				slog.Warnf("Can't parse date (%v): %v", cols[0], err)
				continue
			}

			desc := strings.TrimSpace(cols[2])
			cat := strings.TrimSpace(cols[7])
			if cat != "" {
				desc = "[" + cat + "] " + desc
			}

			deb, err := parseNum(slog, portuguese, cols[3])
			if err != nil {
				continue
			}
			cred, err := parseNum(slog, portuguese, cols[4])
			if err != nil {
				continue
			}
			amt := cred - deb

			amtAfter, err := parseNum(slog, portuguese, cols[5])
			if err != nil {
				continue
			}

			if !yield(&bank_parser.Transaction{
				AuthedAt:            authedAt,
				SettledAt:           settledAt,
				Description:         desc,
				Amt:                 amt,
				AmtAfterTransaction: &amtAfter,
			}) {
				return
			}
		}
	}, nil
}
