package redis

import (
	"os"
	"testing"
	"time"
)

func initRedis() *Redis {
	conf := &RedisConf{}
	conf.Host = os.Getenv("REDIS_HOST")
	if conf.Host == "" {
		conf.Host = "localhost"
	}

	conf.Port = os.Getenv("REDIS_PORT")
	if conf.Port == "" {
		conf.Port = "6379"
	}

	redis, err := NewRedis(conf, "token-test")
	if err != nil {
		panic(err)
	}

	return redis
}

func TestRedisConf(t *testing.T) {
	rc := RedisConf{}

	address := rc.address()
	if address != "localhost:6379" {
		t.Errorf("Expected localhost:6379 as address but got %s", address)
	}

	rc.Host = "192.169.59.103"
	rc.Port = "6380"
	address = rc.address()
	if address != "192.169.59.103:6380" {
		t.Errorf("Expected 192.169.59.103:6380 as address but got %s", address)
	}
}

func TestSetAndGet(t *testing.T) {
	redis := initRedis()
	defer func() {
		err := redis.Del("12")
		if err != nil {
			panic(err)
		}
		redis.Close()
	}()

	err := redis.Set("12", "123123", time.Now().Add(1*time.Minute))
	if err != nil {
		t.Errorf("Expected nil for set but got %s", err)
		t.FailNow()
	}

	token, err := redis.Get("12")
	if err != nil {
		t.Errorf("Expected nil for get but got %s", err)
		t.FailNow()
	}

	if token != "123123" {
		t.Errorf("Expected token was 123123 but got %s", token)
	}

	// time.Sleep(2 * time.Second)
	// token, err = redis.Get("12")
	// if err != nil {
	// 	t.Errorf("Expected nil for get but got %s", err)
	// 	t.FailNow()
	// }

	// if token != "" {
	// 	t.Errorf("Expected token to be expired but got %s", token)
	// }

}

func TestDel(t *testing.T) {
	redis := initRedis()
	defer redis.Close()

	// test when the key/value does not exist
	err := redis.Del("13")
	if err != nil {
		t.Errorf("Expected nil for delete when value does not exist but got %s", err)
		t.FailNow()
	}

	err = redis.Set("13", "123123", time.Now().Add(5*time.Minute))
	if err != nil {
		t.Errorf("Expected nil for set but got %s", err)
		t.FailNow()
	}

	err = redis.Del("13")
	if err != nil {
		t.Errorf("Expected nil for delete but got %s", err)
		t.FailNow()
	}

	token, err := redis.Get("13")
	if err != nil {
		t.Errorf("Expected nil for get but got %s", err)
		t.FailNow()
	}

	if token != "" {
		t.Errorf("Expected empty token after delete but got %s", token)
	}

}

func TestGetExpireAt(t *testing.T) {
	redis := initRedis()
	defer func() {
		err := redis.Del("15")
		if err != nil {
			panic(err)
		}
		redis.Close()
	}()

	_, err := redis.GetExpireAt("14")
	if err != ErrNotFound {
		t.Errorf("Expected not found error for non existing key in GetExpireAt call but got %s", err)
		t.FailNow()
	}

	willExpireAt := time.Now().Add(4 * time.Minute)

	err = redis.Set("15", "123123", willExpireAt)
	if err != nil {
		t.Errorf("Expected nil for set but got %s", err)
		t.FailNow()
	}

	redis.Get("15")

	expireAt, err := redis.GetExpireAt("15")
	if err != nil {
		t.Errorf("Expected nil for GetExpireAt call when key exists but got %s", err)
		t.FailNow()
	}

	if expireAt.IsZero() {
		t.Error("Expected a value for expireAt field but got zero value")
		t.FailNow()
	}

	if !expireAt.Before(willExpireAt) {
		t.Errorf("Expected expiration value was %s but got %s", willExpireAt, expireAt)
	}
}
