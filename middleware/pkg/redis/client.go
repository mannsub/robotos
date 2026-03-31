package redisclient

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	DefaultAddr = "localhost:6379"

	KeyNeoDMState   = "neodm:state"
	KeyNeoDMEmotion = "neodm:emotion"
	KeySensorJoint  = "sensor:joint"
	KeySensorIMU    = "sensor:imu"
	KeyRobotBattery = "robot:battery"

	StateTTL = 5 * time.Second
)

type Client struct {
	rdb *redis.Client
}

func New(addr string) *Client {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Client{rdb: rdb}
}

func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, key, data, ttl).Err()
}

func (c *Client) Get(ctx context.Context, key string, dest any) error {
	data, err := c.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

func (c *Client) Publish(ctx context.Context, channel string, value any) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.rdb.Publish(ctx, channel, data).Err()
}

func (c *Client) Subscribe(ctx context.Context, channel string) *redis.PubSub {
	return c.rdb.Subscribe(ctx, channel)
}

func (c *Client) Ping(ctx context.Context) error {
	return c.rdb.Ping(ctx).Err()
}

func (c *Client) Close() error {
	return c.rdb.Close()
}
