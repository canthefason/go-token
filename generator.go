package token

import (
	"errors"
	"time"

	"github.com/canthefason/go-token/redis"
	uuid "github.com/nu7hatch/gouuid"
)

var (
	ErrIdNotSet     = errors.New("id is not set")
	ErrValueNotSet  = errors.New("token value is not set")
	ErrNotFound     = errors.New("not found")
	ErrInvalidToken = errors.New("invalid token")
)

type TokenManager struct {
	redisConn *redis.Redis
	ttl       time.Duration
}

func NewTokenManager(redisConn *redis.Redis, ttl time.Duration) *TokenManager {
	return &TokenManager{redisConn: redisConn, ttl: ttl}
}

// Authenticate checks if token is valid for the given account
func (tm *TokenManager) Authenticate(t *Token) error {
	if t.Id == "" {
		return ErrIdNotSet
	}
	if t.Value == "" {
		return ErrValueNotSet
	}

	token, err := tm.Get(t.Id)
	if err == ErrNotFound {
		return ErrInvalidToken
	}

	if err != nil {
		return err
	}

	if token.Value != t.Value {
		return ErrInvalidToken
	}

	return nil
}

// GetOrCreate gets token of the given account, and creates if it
// does not exist
func (tm *TokenManager) GetOrCreate(id string) (*Token, error) {
	if id == "" {
		return nil, ErrIdNotSet
	}

	token, err := tm.Get(id)
	if err == nil {
		return token, nil
	}

	if err != ErrNotFound {
		return nil, err
	}

	return tm.Create(id)
}

func (tm *TokenManager) Invalidate(id string) error {
	if id == "" {
		return ErrIdNotSet
	}

	return tm.redisConn.Del(id)
}

func (tm *TokenManager) Get(id string) (*Token, error) {
	tokenValue, err := tm.redisConn.Get(id)
	if err != nil {
		return nil, err
	}

	if tokenValue == "" {
		return nil, ErrNotFound
	}

	expireAt, err := tm.redisConn.GetExpireAt(id)
	if err != nil {
		return nil, err
	}

	token := NewToken()

	token.Id = id
	token.Value = tokenValue
	token.ExpireAt = expireAt.Round(time.Minute)

	return token, nil
}

// Create creates a token with a random uuid and TTL. It is not idempotent,
// therefore with every call, it regenerates the token for id
func (tm *TokenManager) Create(id string) (*Token, error) {
	if id == "" {
		return nil, ErrIdNotSet
	}

	tokenValue, err := tm.generateUUID()
	if err != nil {
		return nil, err
	}

	// create token
	expireAt := time.Now().Add(tm.ttl).Round(time.Minute)

	token := NewToken()
	token.Id = id
	token.Value = tokenValue
	token.ExpireAt = expireAt

	if err := tm.redisConn.Set(id, tokenValue, expireAt); err != nil {
		return nil, err
	}

	return token, nil
}

// generateUUID creates a version 4 random uuid
func (tm *TokenManager) generateUUID() (string, error) {
	uuid4, err := uuid.NewV4()
	if err != nil {
		return "", err
	}

	return uuid4.String(), nil
}
