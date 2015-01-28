package token

import "time"

type Token struct {
	// unique id for token owner
	Id string

	// token value
	Value string

	// expiration time
	ExpireAt time.Time
}

func NewToken() *Token {
	return &Token{}
}
