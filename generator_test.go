package token

import (
	"os"
	"testing"
	"time"

	"github.com/canthefason/go-token/redis"
)

func tearUp(t *testing.T) *TokenManager {
	conf := &redis.RedisConf{}
	conf.Host = os.Getenv("REDIS_HOST")
	if conf.Host == "" {
		conf.Host = "localhost"
	}

	conf.Port = os.Getenv("REDIS_PORT")
	if conf.Port == "" {
		conf.Port = "6379"
	}

	redisConn, err := redis.NewRedis(conf, "generator-testing")
	if err != nil {
		t.Fatalf("Could not initialize redis: %s", err)
	}

	return NewTokenManager(redisConn, 1*time.Minute)
}

func TestGenerateUUID(t *testing.T) {
	tm := tearUp(t)
	defer tm.redisConn.Close()

	uuid, err := tm.generateUUID()
	if err != nil {
		t.Errorf("Unexpected error in uuid generation: %s", err)
	}
	if len(uuid) != 36 {
		t.Errorf("Unexpected format for uuid")
	}

}

func TestTokenSave(t *testing.T) {
	tm := tearUp(t)
	defer tm.redisConn.Close()

	_, err := tm.Create("")
	if err == nil {
		t.Errorf("Expected 'id is not set' error but got nil while creating token")
	}

	token, err := tm.Create("3")
	if err != nil {
		t.Errorf("Expected nil but got %s in token creation", err)
		t.FailNow()
	}

	defer tm.redisConn.Del("3")

	tokenValue, err := tm.redisConn.Get("3")
	if err != nil {
		t.Errorf("Expected token value but got %s", err)
	}

	if tokenValue != token.Value {
		t.Errorf("Expected %s as token but got %s", token.Value, tokenValue)
	}

}

func TestTokenGet(t *testing.T) {
	tm := tearUp(t)
	defer tm.redisConn.Close()

	_, err := tm.Get("4")
	if err != ErrNotFound {
		t.Error("Expected 'not found' error, but got %s", err)
	}

	err = tm.redisConn.Set("5", "1467", time.Now().Add(1*time.Minute).Round(time.Minute))
	if err != nil {
		t.Errorf("Expected nil but got %s while creating token in cache", err)
		t.FailNow()
	}
	defer tm.redisConn.Del("5")

	token, err := tm.Get("5")
	if err != nil {
		t.Errorf("Expected nil but got %s while fetching token", err)
		t.FailNow()
	}

	if token.Id != "5" {
		t.Errorf("Expected '5' as id but got %s", token.Id)
	}

	if token.Value != "1467" {
		t.Errorf("Expected '1467' as token value but got %s", token.Value)
	}

	expectedTime := time.Now().Add(1 * time.Minute).Round(time.Minute)
	if expectedTime.After(token.ExpireAt) ||
		!expectedTime.Equal(token.ExpireAt) {
		t.Errorf("Invalid ExpireAt value of token: %s", token.ExpireAt)
	}
}

func TestTokenInvalidate(t *testing.T) {
	tm := tearUp(t)
	defer tm.redisConn.Close()

	err := tm.Invalidate("")
	if err == nil {
		t.Error("Expected 'id is not set' error but got nil")
	}

	err = tm.redisConn.Set("6", "1467", time.Now().Add(1*time.Minute).Round(time.Minute))
	if err != nil {
		t.Errorf("Expected nil but got %s while creating token in cache", err)
		t.FailNow()
	}
	defer tm.redisConn.Del("6")

	err = tm.Invalidate("6")
	if err != nil {
		t.Errorf("Unexpected error while invalidating token: %s", err)
	}

	token, err := tm.redisConn.Get("6")
	if err != nil {
		t.Errorf("Unexpected error %s", err)
	}

	if token != "" {
		t.Errorf("Expected empty token after token invalidation, but got %s", token)
	}

}

func TestTokenGetOrCreate(t *testing.T) {
	tm := tearUp(t)
	defer tm.redisConn.Close()

	_, err := tm.GetOrCreate("")
	if err == nil {
		t.Error("Expected 'id is not set' error but got nil")
	}

	err = tm.redisConn.Set("7", "1467", time.Now().Add(1*time.Minute).Round(time.Minute))
	if err != nil {
		t.Errorf("Expected nil but got %s while creating token in cache", err)
		t.FailNow()
	}
	defer tm.redisConn.Del("7")

	token, err := tm.GetOrCreate("7")
	if err != nil {
		t.Errorf("Unexpected error while getting the token from cache: %s", err)
	}

	if token.Value != "1467" {
		t.Errorf("Expected '1467' as token value but got %s", token.Value)
	}

	token, err = tm.GetOrCreate("8")
	if err != nil {
		t.Errorf("Unexpected error while creating the token in cache: %s", err)
	}
	defer tm.redisConn.Del("8")

	if token.Value == "" {
		t.Errorf("Expected a token value but got empty string")
	}
}

func TestTokenAuthenticate(t *testing.T) {
	tm := tearUp(t)
	defer tm.redisConn.Close()

	token := NewToken()
	err := tm.Authenticate(token)
	if err != ErrIdNotSet {
		t.Errorf("Expected 'id is not set' error but got %s", err)
	}
	token.Id = "9"

	err = tm.Authenticate(token)
	if err != ErrValueNotSet {
		t.Errorf("Expected 'token value is not set' error but got %s", err)
	}
	token.Value = "3579"

	err = tm.Authenticate(token)
	if err != ErrInvalidToken {
		t.Errorf("Expected 'token value is invalid' error but got %s", err)
	}

	err = tm.redisConn.Set("9", "3578", time.Now().Add(1*time.Minute).Round(time.Minute))
	if err != nil {
		t.Errorf("Expected nil but got %s while creating token in cache", err)
		t.FailNow()
	}
	defer tm.redisConn.Del("9")

	err = tm.Authenticate(token)
	if err != ErrInvalidToken {
		t.Errorf("Expected 'token value is invalid' error but got %s", err)
	}

	token.Value = "3578"
	err = tm.Authenticate(token)
	if err != nil {
		t.Errorf("Unexpected error in token authentication: %s", err)
	}

}
