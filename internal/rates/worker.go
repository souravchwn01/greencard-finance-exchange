package rates

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type WorkerConfig struct {
	ConsumerName      string
	PoolSize          int
	ReadCount         int64
	ReadBlock         time.Duration
	ClaimMinIdle      time.Duration
	ProcessedEventTTL time.Duration
}

type Processor struct {
	log    *zap.Logger
	stream *Stream
	store  *Store
	cache  *Cache
	rdb    *redis.Client
	cfg    WorkerConfig
}

func NewProcessor(log *zap.Logger, stream *Stream, store *Store, cache *Cache, rdb *redis.Client, cfg WorkerConfig) *Processor {
	return &Processor{
		log:    log,
		stream: stream,
		store:  store,
		cache:  cache,
		rdb:    rdb,
		cfg:    cfg,
	}
}

type streamJob struct {
	id     string
	values map[string]any
}

func (p *Processor) Run(ctx context.Context) error {
	if p.cfg.PoolSize <= 0 {
		return fmt.Errorf("invalid worker pool size: %d", p.cfg.PoolSize)
	}
	if p.cfg.ReadCount <= 0 {
		p.cfg.ReadCount = 10
	}
	if p.cfg.ReadBlock <= 0 {
		p.cfg.ReadBlock = 2 * time.Second
	}
	if p.cfg.ClaimMinIdle <= 0 {
		p.cfg.ClaimMinIdle = 30 * time.Second
	}
	if p.cfg.ProcessedEventTTL <= 0 {
		p.cfg.ProcessedEventTTL = 24 * time.Hour
	}

	jobs := make(chan streamJob, p.cfg.PoolSize*4)

	var wg sync.WaitGroup
	for i := 0; i < p.cfg.PoolSize; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			p.workerLoop(ctx, workerID, jobs)
		}(i + 1)
	}

	readErr := make(chan error, 1)
	go func() {
		readErr <- p.consumeLoop(ctx, jobs)
	}()

	<-ctx.Done()
	close(jobs)
	wg.Wait()

	select {
	case err := <-readErr:
		if err != nil && ctx.Err() == nil {
			return err
		}
	default:
	}
	return nil
}

func (p *Processor) consumeLoop(ctx context.Context, jobs chan<- streamJob) error {
	// 1) Normal new messages (>)
	// 2) Reclaim stuck messages with XAUTOCLAIM in a separate goroutine so it doesn't
	// block on the XREADGROUP call.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		claimStart := "0-0"
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				msgs, next, err := p.stream.AutoClaim(ctx, p.cfg.ConsumerName, p.cfg.ClaimMinIdle, claimStart, p.cfg.ReadCount)
				if err != nil {
					p.log.Warn("xautoclaim failed", zap.Error(err))
					continue
				}
				claimStart = next
				for _, m := range msgs {
					select {
					case jobs <- streamJob{id: m.ID, values: m.Values}:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		msgs, err := p.stream.ReadGroup(ctx, p.cfg.ConsumerName, p.cfg.ReadCount, p.cfg.ReadBlock)
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			p.log.Error("xreadgroup failed", zap.Error(err))
			time.Sleep(500 * time.Millisecond)
			continue
		}

		for _, m := range msgs {
			select {
			case jobs <- streamJob{id: m.ID, values: m.Values}:
			case <-ctx.Done():
				return nil
			}
		}
	}
}

func (p *Processor) workerLoop(ctx context.Context, workerID int, jobs <-chan streamJob) {
	for job := range jobs {
		if ctx.Err() != nil {
			return
		}

		if err := p.processOne(ctx, job); err != nil {
			p.log.Error("rate event processing failed", zap.Int("worker_id", workerID), zap.String("stream_id", job.id), zap.Error(err))
			// Do NOT ack on failure; message will be retried / reclaimed.
			continue
		}
	}
}

func (p *Processor) processOne(ctx context.Context, job streamJob) error {
	ev, err := parseRateEvent(job.values)
	if err != nil {
		// Bad payload: ack to avoid poison pill loop.
		_ = p.stream.Ack(ctx, job.id)
		return fmt.Errorf("invalid stream payload (acked): %w", err)
	}

	// Idempotency: processed:{event_id}
	processedKey := "processed:" + ev.EventID
	ok, err := p.rdb.SetNX(ctx, processedKey, "1", p.cfg.ProcessedEventTTL).Result()
	if err != nil {
		return fmt.Errorf("setnx idempotency: %w", err)
	}
	if !ok {
		// Already processed: ACK and stop.
		_ = p.stream.Ack(ctx, job.id)
		return nil
	}

	if err := p.store.InsertEvent(ctx, ev); err != nil {
		return err
	}

	if err := p.cache.SetLatest(ctx, CachedRate{
		Pair:       ev.Pair,
		Rate:       ev.Rate,
		ProviderID: ev.ProviderID,
		UpdatedAt:  ev.Timestamp.UTC(),
	}); err != nil {
		return err
	}

	if err := p.stream.Ack(ctx, job.id); err != nil {
		return err
	}
	return nil
}

func parseRateEvent(values map[string]any) (RateEvent, error) {
	get := func(key string) (string, error) {
		v, ok := values[key]
		if !ok {
			return "", fmt.Errorf("missing field %q", key)
		}
		switch t := v.(type) {
		case string:
			return t, nil
		case []byte:
			return string(t), nil
		default:
			return fmt.Sprintf("%v", t), nil
		}
	}

	eventID, err := get("event_id")
	if err != nil {
		return RateEvent{}, err
	}
	providerID, err := get("provider_id")
	if err != nil {
		return RateEvent{}, err
	}
	pair, err := get("pair")
	if err != nil {
		return RateEvent{}, err
	}
	rateStr, err := get("rate")
	if err != nil {
		return RateEvent{}, err
	}
	rate, err := strconv.ParseFloat(rateStr, 64)
	if err != nil {
		return RateEvent{}, fmt.Errorf("invalid rate %q", rateStr)
	}
	tsStr, err := get("timestamp")
	if err != nil {
		return RateEvent{}, err
	}
	ts, err := time.Parse(time.RFC3339Nano, tsStr)
	if err != nil {
		// accept RFC3339 too
		ts2, err2 := time.Parse(time.RFC3339, tsStr)
		if err2 != nil {
			return RateEvent{}, fmt.Errorf("invalid timestamp %q", tsStr)
		}
		ts = ts2
	}

	pair = NormalizePair(pair)
	if err := ValidatePair(pair); err != nil {
		return RateEvent{}, err
	}
	if rate <= 0 {
		return RateEvent{}, fmt.Errorf("rate must be positive")
	}

	return RateEvent{
		EventID:    eventID,
		ProviderID: providerID,
		Pair:       pair,
		Rate:       rate,
		Timestamp:  ts.UTC(),
	}, nil
}
