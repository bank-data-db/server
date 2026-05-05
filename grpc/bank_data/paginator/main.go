package paginator

type PaginationRequest interface {
	GetPageSize() uint32
	HasPageSize() bool
	GetPaginationToken() string
	HasPaginationToken() bool
}

type PaginationResponse[R any] interface {
	SetTotalCount(uint32)
	SetPaginationToken(string)
	SetResult([]R)
}

type PaginationResponseValue interface {
	GetID() string
}
