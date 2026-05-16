package slogctx

import (
	"context"
	"iter"
	"log/slog"
	"slices"
)

// Oh yeah this isn't fragile at all fuck you

type ctxKeyT int

const ctxKey ctxKeyT = 0

func valIter(vals []any) iter.Seq[slog.Attr] {
	i := 0

	return func(yield func(slog.Attr) bool) {
		k := vals[i]
		switch v := k.(type) {
		case slog.Attr:
			if !yield(v) {
				return
			}
			i++
		case string:
			if !yield(slog.Any(v, vals[i+1])) {
				return
			}
			i += 2
		default:
			panic("slogctx.With provided with a wrong argument count/type!")
		}
	}
}

func With(ctx context.Context, vals ...any) context.Context {
	if len(vals)%2 != 0 {
		panic("slogctx.Enrich called with an un-even val set")
	}

	valSeq := valIter(vals)
	existing := ctx.Value(ctxKey)
	if existing == nil {
		return context.WithValue(ctx, ctxKey, slices.Collect(valSeq))
	}

	arr := slices.Grow(existing.([]slog.Attr), len(vals)/2)

	for a := range valSeq {
		dupe := false
		for _, existingA := range arr {
			if existingA.Key == a.Key {
				dupe = true
				break
			}
		}

		if !dupe {
			arr = append(arr, a)
		}
	}

	return context.WithValue(ctx, ctxKey, arr)
}

type ctxHandler struct {
	attrs []slog.Attr
	real  slog.Handler
}

// Enabled implements [slog.Handler].
func (c *ctxHandler) Enabled(ctx context.Context, l slog.Level) bool {
	return c.real.Enabled(ctx, l)
}

// Handle implements [slog.Handler].
func (c *ctxHandler) Handle(ctx context.Context, r slog.Record) error {
	existing := ctx.Value(ctxKey)
	if existing == nil {
		return c.real.Handle(ctx, r)
	}

	arr := existing.([]slog.Attr)
	for _, ctxAttr := range arr {
		dupe := false
		for _, loggerAttr := range c.attrs {
			if loggerAttr.Key == ctxAttr.Key {
				dupe = true
				break
			}
		}

		if !dupe {
			r.AddAttrs(ctxAttr)
		}
	}

	return c.real.Handle(ctx, r)
}

// WithAttrs implements [slog.Handler].
func (c *ctxHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	c2 := &ctxHandler{
		attrs: make([]slog.Attr, len(c.attrs)+len(attrs)),
		real:  c.real.WithAttrs(attrs),
	}
	copied := copy(c2.attrs, c.attrs)
	copy(c2.attrs[copied:], attrs)

	return c2
}

// WithGroup implements [slog.Handler].
func (c *ctxHandler) WithGroup(name string) slog.Handler {
	return &ctxHandler{
		attrs: nil,
		real:  c.real.WithGroup(name),
	}
}

func NewHandler(real slog.Handler) slog.Handler {
	return &ctxHandler{
		attrs: nil,
		real:  real,
	}
}

func Logger(base *slog.Logger, ctx context.Context) *slog.Logger {
	existing := ctx.Value(ctxKey)
	if existing == nil {
		return base
	}

	return base.With(existing.([]any)...)
}
