package external

import (
	"bufio"
	nerr "errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/shadiestgoat/bankDataDB/bank_parser"
	"github.com/shadiestgoat/bankDataDB/external/errors"
	"github.com/shadiestgoat/bankDataDB/internal"
)

func mountUpload(api *internal.API, r chi.Router) {
	defHTTP(r, `POST`, `/`, api, func(r *http.Request) (any, errors.GenericHTTPError) {
		defer r.Body.Close()

		body := bufio.NewReader(r.Body)

		log := api.Logger()(r.Context())

		iter, err := bank_parser.Iter(r.Context(), body)
		if err != nil {
			if !nerr.Is(err, bank_parser.ErrAmbiguous) {
				log.Errorw("Error when parsing bank sheet", "error", err)
			} else {
				log.Debugw("Ambiguous bank sheet")
			}

			return nil, errors.BadBankSheet
		}
		if iter == nil {
			log.Debugw("Unknown bank sheet")
			return nil, errors.BadBankSheet
		}

		resp, err := api.UploadBankIter(r.Context(), iter, getUserID(r))
		if err != nil {
			if err, ok := err.(errors.GenericHTTPError); ok {
				return nil, err
			}

			return nil, errors.InternalErr
		}

		return resp, nil
	})
}
