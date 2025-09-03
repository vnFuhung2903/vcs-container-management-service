package workers

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/vnFuhung2903/vcs-container-management-service/dto"
	"github.com/vnFuhung2903/vcs-container-management-service/interfaces"
	"github.com/vnFuhung2903/vcs-container-management-service/pkg/logger"
	"github.com/vnFuhung2903/vcs-container-management-service/usecases/repositories"
	"go.uber.org/zap"
)

type IConsumerWorker interface {
	Start()
	Stop()
}

type consumerWorker struct {
	kafkaConsumer interfaces.IKafkaConsumer
	redisClient   interfaces.IRedisClient
	repo          repositories.IContainerRepository
	logger        logger.ILogger
	ctx           context.Context
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
}

func NewConsumerWorker(
	kafkaConsumer interfaces.IKafkaConsumer,
	redisClient interfaces.IRedisClient,
	repo repositories.IContainerRepository,
	logger logger.ILogger,
) IConsumerWorker {
	ctx, cancel := context.WithCancel(context.Background())
	return &consumerWorker{
		kafkaConsumer: kafkaConsumer,
		redisClient:   redisClient,
		repo:          repo,
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		wg:            &sync.WaitGroup{},
	}
}

func (w *consumerWorker) Start() {
	w.wg.Add(1)

	go func() {
		defer w.wg.Done()

		for {
			select {
			case <-w.ctx.Done():
				return
			default:
				w.consume()
			}
		}
	}()
}

func (w *consumerWorker) Stop() {
	w.cancel()
	w.wg.Wait()
	w.logger.Info("consumer worker stopped")
}

func (w *consumerWorker) consume() {
	message, err := w.kafkaConsumer.Consume(w.ctx)
	if err != nil {
		w.logger.Error("failed to consume message", zap.Error(err))
		return
	}

	updateStatus := dto.KafkaStatusUpdate{}
	if err := json.Unmarshal(message.Value, &updateStatus); err != nil {
		w.logger.Error("failed to unmarshal message", zap.Error(err))
		return
	}

	containers, err := w.redisClient.Get(w.ctx, "containers")
	if err != nil {
		w.logger.Error("failed to retrieve containers from redis", zap.Error(err))
		return
	}

	for i := range containers {
		if containers[i].ContainerId == updateStatus.ContainerId {
			containers[i].Status = updateStatus.Status
			break
		}
	}

	if err := w.redisClient.Set(w.ctx, "containers", containers); err != nil {
		w.logger.Error("failed to update containers in redis", zap.Error(err))
		return
	}

	if err := w.repo.Update(updateStatus.ContainerId, updateStatus.Status, updateStatus.Ipv4); err != nil {
		w.logger.Error("failed to update container", zap.Error(err))
		return
	}

	w.logger.Info("consumed message from kafka successfully", zap.String("container_id", updateStatus.ContainerId), zap.String("status", string(updateStatus.Status)))
}
