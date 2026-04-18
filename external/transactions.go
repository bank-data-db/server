package external

import (
	// "context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shadiestgoat/bankDataDB/external/errors"
	"github.com/shadiestgoat/bankDataDB/internal"
)

// For /transactions
func mountTransactions(a *internal.API, r chi.Router) {
	defHTTPPage(
		r, a,
		[]string{"authed_at", "settled_at", "amount", "category"},
		func(r *http.Request, size, off int, orderBy string, asc bool) (any, int, errors.GenericHTTPError) {
			res, err := a.GetTransactions(r.Context(), getUserID(r), size, off, internal.TransactionOrderBy(orderBy), asc)
			if err != nil {
				return nil, 0, errors.InternalErr
			}
			c, err := a.GetTransactionsCount(r.Context(), getUserID(r))
			if err != nil {
				return nil, 0, errors.InternalErr
			}

			return res, c, nil
		},
	)

	// r.Route(`/{id}`, func(r chi.Router) {
	// 	r.Use(makeMiddleware(a, func(a *internal.API, r *http.Request) (*http.Request, error) {
	// 		m, err := a.MappingGetByID(r.Context(), getUserID(r), chi.URLParam(r, "id"))
	// 		if err != nil {
	// 			return nil, err
	// 		}
	// 		if m == nil {
	// 			return nil, errors.NotFound
	// 		}

	// 		return r.WithContext(context.WithValue(r.Context(), CTX_MAPPING, m)), nil
	// 	}))

	// 	defHTTPRead(r, `GET`, `/`)
	// })
}
