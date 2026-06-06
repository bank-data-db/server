package paginator

import (
	"strconv"
	"time"
)

func ColCfgFloat[RV PaginationResponseValue](dbName string, getter func(v RV) float64) *ColCfg[RV] {
	return &ColCfg[RV]{
		DBName: dbName,
		Marshal: func(v RV) string {
			return strconv.FormatFloat(getter(v), 'f', -1, 64)
		},
		Unmarshall: func(v string) (any, error) {
			return strconv.ParseFloat(v, 64)
		},
	}
}

func ColCfgUnixMilli[RV PaginationResponseValue](dbName string, getter func(v RV) int64) *ColCfg[RV] {
	return &ColCfg[RV]{
		DBName: dbName,
		Marshal: func(v RV) string {
			return strconv.FormatInt(getter(v), 10)
		},
		Unmarshall: func(v string) (any, error) {
			t, err := strconv.ParseInt(v, 10, 64)
			if err != nil {
				return nil, err
			}

			return time.UnixMilli(t), nil
		},
	}
}
