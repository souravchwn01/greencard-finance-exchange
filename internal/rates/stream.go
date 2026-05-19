package rates

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

type Stream struct {
	rdb        *redis.Client
	streamName string
	groupName  string
}

func NewStream(rdb *redis.Client, streamName, groupName string) *Stream {
	return &Stream{
		rdb:        rdb,
		streamName: streamName,
		groupName:  groupName,
	}
}

func (s *Stream) EnsureConsumerGroup(ctx context.Context) error {
	// Create group and stream if not exists.
	err := s.rdb.XGroupCreateMkStream(ctx, s.streamName, s.groupName, "0").Err()
	if err == nil {
		return nil
	}
	// BUSYGROUP means it already exists.
	if isBusyGroupErr(err) {
		return nil
	}
	return fmt.Errorf("create consumer group: %w", err)
}

func (s *Stream) Enqueue(ctx context.Context, ev RateEvent) error {
	args := &redis.XAddArgs{
		Stream: s.streamName,
		Values: map[string]any{
			"event_id":    ev.EventID,
			"provider_id": ev.ProviderID,
			"pair":        NormalizePair(ev.Pair),
			"rate":        strconv.FormatFloat(ev.Rate, 'f', -1, 64),
			"timestamp":   ev.Timestamp.UTC().Format(time.RFC3339Nano),
		},
	}
	if err := s.rdb.XAdd(ctx, args).Err(); err != nil {
		return fmt.Errorf("xadd: %w", err)
	}
	return nil
}

func (s *Stream) ReadGroup(ctx context.Context, consumer string, count int64, block time.Duration) ([]redis.XMessage, error) {
	res, err := s.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    s.groupName,
		Consumer: consumer,
		Streams:  []string{s.streamName, ">"},
		Count:    count,
		Block:    block,
		NoAck:    false,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, fmt.Errorf("xreadgroup: %w", err)
	}
	if len(res) == 0 {
		return nil, nil
	}
	return res[0].Messages, nil
}

func (s *Stream) Ack(ctx context.Context, ids ...string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.rdb.XAck(ctx, s.streamName, s.groupName, ids...).Err(); err != nil {
		return fmt.Errorf("xack: %w", err)
	}
	return nil
}

func (s *Stream) AutoClaim(ctx context.Context, consumer string, minIdle time.Duration, start string, count int64) ([]redis.XMessage, string, error) {
	// XAUTOCLAIM returns next start ID + messages.
	msgs, nextStart, err := s.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   s.streamName,
		Group:    s.groupName,
		Consumer: consumer,
		MinIdle:  minIdle,
		Start:    start,
		Count:    count,
	}).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, start, nil
		}
		return nil, start, fmt.Errorf("xautoclaim: %w", err)
	}
	return msgs, nextStart, nil
}

func isBusyGroupErr(err error) bool {
	// go-redis doesn't export a sentinel; check message.
	return err != nil && (strings.Contains(err.Error(), "BUSYGROUP") || strings.Contains(err.Error(), "Consumer Group name already exists"))
}
