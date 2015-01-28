package redis

import (
	"errors"
	"fmt"
	"time"

	"github.com/garyburd/redigo/redis"
)

type Redis struct {
	conn   redis.Conn
	prefix string
}

var (
	ErrNotFound     = errors.New("not found")
	ErrInvalidValue = errors.New("invalid value")
)

const Prefix = "go-token"

func NewRedis(conf *RedisConf, prefix string) (*Redis, error) {
	c, err := redis.Dial("tcp", conf.address())
	if err != nil {
		return nil, err
	}

	if conf.DB != 0 {
		if _, err := c.Do("SELECT", conf.DB); err != nil {
			c.Close()
			return nil, err
		}
	}

	redis := &Redis{conn: c, prefix: prefix}

	if redis.prefix == "" {
		redis.prefix = Prefix
	}

	return redis, nil
}

// Set sets the token for the given id and also sets expiration time.
func (r *Redis) Set(id string, token string, expireAt time.Time) error {
	key := r.prepareKey(id)

	r.conn.Send("MULTI")
	r.conn.Send("SET", key, token)
	r.conn.Send("EXPIREAT", key, expireAt.Unix())
	_, err := r.conn.Do("EXEC")
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves the token with given id
func (r *Redis) Get(id string) (string, error) {
	key := r.prepareKey(id)

	reply, err := redis.String(r.conn.Do("GET", key))
	if err != nil {
		if err == redis.ErrNil {
			return "", nil
		}

		return "", err
	}

	return reply, nil
}

// Del deletes the token with given id
func (r *Redis) Del(id string) error {
	key := r.prepareKey(id)

	_, err := r.conn.Do("DEL", key)
	if err != nil {
		return err
	}

	return nil
}

// GetExpireAt returns expiration time of the token for the given id
func (r *Redis) GetExpireAt(id string) (time.Time, error) {
	key := r.prepareKey(id)

	res, err := redis.Int(r.conn.Do("TTL", key))
	if err != nil {
		return time.Time{}, err
	}

	if res == -2 {
		return time.Time{}, ErrNotFound
	}

	if res == -1 {
		return time.Time{}, nil
	}

	// TODO it is interesting that whatever I do it gives +4 seconds of the expected ttl
	expireAt, err := time.ParseDuration(fmt.Sprintf("%ds", res-4))
	if err != nil {
		return time.Time{}, ErrInvalidValue
	}

	return time.Now().Add(expireAt), nil
}

func (r *Redis) prepareKey(id string) string {
	return fmt.Sprintf("%s:%s", r.prefix, id)
}

func (r *Redis) Close() {
	r.conn.Close()
}

///////////// RedisConf //////////////

type RedisConf struct {
	// Host name of the redis server
	Host string

	// Redis server Port
	Port string

	// Redis DB to use
	DB int
}

func (rc RedisConf) address() string {
	if rc.Host == "" {
		rc.Host = "localhost"
	}

	if rc.Port == "" {
		rc.Port = "6379"
	}

	return fmt.Sprintf("%s:%s", rc.Host, rc.Port)
}
