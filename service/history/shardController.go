// Copyright (c) 2017 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

package history

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/uber/cadence/.gen/go/shared"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/membership"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/service"
	"github.com/uber/cadence/service/worker/replicator"
)

const (
	shardControllerMembershipUpdateListenerName = "ShardController"
)

const (
	shardsItemStatusInitialized shardsItemStatus = iota
	shardsItemStatusStarted
	shardsItemStatusStopped
)

type (
	shardsItemStatus int

	shardController struct {
		status int32

		service  service.Service
		hostInfo *membership.HostInfo
		config   *Config

		historyServiceResolver membership.ServiceResolver
		metricsClient          metrics.Client
		logger                 log.Logger
		throttledLogger        log.Logger

		historyEventNotifier    historyEventNotifier
		messagePublisher        messaging.Producer
		domainReplicator        replicator.DomainReplicator
		replicationTaskFetchers *ReplicationTaskFetchers

		membershipUpdateCh chan *membership.ChangedEvent
		shardClosedCh      chan int
		shutdownWG         sync.WaitGroup
		shutdownCh         chan struct{}

		sync.RWMutex
		historyShards map[int]*historyShardsItem
	}

	historyShardsItem struct {
		shardID  int
		service  service.Service
		hostInfo *membership.HostInfo
		config   *Config

		historyEventNotifier    historyEventNotifier
		messagePublisher        messaging.Producer
		domainReplicator        replicator.DomainReplicator
		replicationTaskFetchers *ReplicationTaskFetchers

		logger          log.Logger
		throttledLogger log.Logger

		sync.RWMutex
		status shardsItemStatus
		engine Engine
	}
)

func newShardController(
	serviceBase service.Service,
	config *Config,
) (*shardController, error) {

	var err error
	var messagePublisher messaging.Producer
	if serviceBase.GetClusterMetadata().IsGlobalDomainEnabled() {
		messagePublisher, err = serviceBase.GetMessagingClient().NewProducerWithClusterName(
			serviceBase.GetClusterMetadata().GetCurrentClusterName(),
		)
		return nil, err
	}

	hostInfo, err := serviceBase.GetHostInfo()
	if err != nil {
		return nil, err
	}

	return &shardController{
		status: common.DaemonStatusInitialized,

		service:  serviceBase,
		hostInfo: hostInfo,
		config:   config,

		historyServiceResolver: serviceBase.GetHistoryServiceResolver(),
		metricsClient:          serviceBase.GetMetricsClient(),
		logger:                 serviceBase.GetLogger().WithTags(tag.ComponentShardController),
		throttledLogger:        serviceBase.GetThrottledLogger().WithTags(tag.ComponentShardController),

		historyEventNotifier: newHistoryEventNotifier(
			serviceBase.GetTimeSource(),
			serviceBase.GetMetricsClient(),
			config.GetShardID,
		),
		messagePublisher: messagePublisher,
		domainReplicator: replicator.NewDomainReplicator(
			serviceBase.GetMetadataManager(),
			serviceBase.GetLogger(),
		),
		replicationTaskFetchers: NewReplicationTaskFetchers(serviceBase),

		membershipUpdateCh: make(chan *membership.ChangedEvent, 10),
		shardClosedCh:      make(chan int, config.NumberOfShards),
		shutdownCh:         make(chan struct{}),

		historyShards: make(map[int]*historyShardsItem),
	}, nil
}

func newHistoryShardsItem(
	shardID int,
	serviceBase service.Service,
	config *Config,
	hostInfo *membership.HostInfo,
	historyEventNotifier historyEventNotifier,
	messagePublisher messaging.Producer,
	domainReplicator replicator.DomainReplicator,
	replicationTaskFetchers *ReplicationTaskFetchers,
	logger log.Logger,
	throttledLogger log.Logger,
) (*historyShardsItem, error) {

	return &historyShardsItem{
		shardID: shardID,
		status:  shardsItemStatusInitialized,

		service:  serviceBase,
		hostInfo: hostInfo,
		config:   config,

		historyEventNotifier:    historyEventNotifier,
		messagePublisher:        messagePublisher,
		domainReplicator:        domainReplicator,
		replicationTaskFetchers: replicationTaskFetchers,

		logger:          logger.WithTags(tag.ShardID(shardID)),
		throttledLogger: throttledLogger.WithTags(tag.ShardID(shardID)),
	}, nil
}

func (c *shardController) Start() {
	if !atomic.CompareAndSwapInt32(&c.status, common.DaemonStatusInitialized, common.DaemonStatusStarted) {
		return
	}

	c.historyEventNotifier.Start()
	c.replicationTaskFetchers.Start()

	c.acquireShards()
	c.shutdownWG.Add(1)
	go c.shardManagementEventLoop()

	if err := c.historyServiceResolver.AddListener(
		shardControllerMembershipUpdateListenerName,
		c.membershipUpdateCh,
	); err != nil {
		c.logger.Fatal("unable to start shard controller", tag.Error(err))
	}

	c.logger.Info("shard controller started", tag.LifeCycleStarted, tag.Address(c.hostInfo.Identity()))
}

func (c *shardController) Stop() {
	if !atomic.CompareAndSwapInt32(&c.status, common.DaemonStatusStarted, common.DaemonStatusStopped) {
		return
	}

	if err := c.historyServiceResolver.RemoveListener(
		shardControllerMembershipUpdateListenerName,
	); err != nil {
		c.logger.Error("error removing membership update listener", tag.Error(err), tag.OperationFailed)
	}
	close(c.shutdownCh)

	if success := common.AwaitWaitGroup(&c.shutdownWG, time.Minute); !success {
		c.logger.Warn("shard controller stop timeout", tag.LifeCycleStopTimedout, tag.Address(c.hostInfo.Identity()))
	}

	c.replicationTaskFetchers.Stop()
	c.historyEventNotifier.Stop()
	c.logger.Info("shard controller stopped", tag.LifeCycleStopped, tag.Address(c.hostInfo.Identity()))
}

func (c *shardController) GetEngine(
	workflowID string,
) (Engine, error) {

	shardID := c.config.GetShardID(workflowID)
	return c.getEngineForShard(shardID)
}

func (c *shardController) getEngineForShard(
	shardID int,
) (Engine, error) {

	sw := c.metricsClient.StartTimer(metrics.HistoryShardControllerScope, metrics.GetEngineForShardLatency)
	defer sw.Stop()
	item, err := c.getOrCreateHistoryShardItem(shardID)
	if err != nil {
		return nil, err
	}
	return item.getOrCreateEngine(c.shardClosedCh)
}

func (c *shardController) removeEngineForShard(
	shardID int,
) {

	sw := c.metricsClient.StartTimer(metrics.HistoryShardControllerScope, metrics.RemoveEngineForShardLatency)
	defer sw.Stop()
	item, _ := c.removeHistoryShardItem(shardID)
	if item != nil {
		item.stopEngine()
	}
}

func (c *shardController) getOrCreateHistoryShardItem(
	shardID int,
) (*historyShardsItem, error) {

	if atomic.LoadInt32(&c.status) != common.DaemonStatusStarted {
		return nil, &shared.InternalServiceError{
			Message: fmt.Sprintf("shardController for host '%v' is not started", c.hostInfo.Identity()),
		}
	}

	c.RLock()
	if item, ok := c.historyShards[shardID]; ok {
		if item.isValid() {
			c.RUnlock()
			return item, nil
		}
		// if item not valid then process to create a new one
	}
	c.RUnlock()

	c.Lock()
	defer c.Unlock()

	if item, ok := c.historyShards[shardID]; ok && item.isValid() {
		return item, nil
	}

	info, err := c.historyServiceResolver.Lookup(string(shardID))
	if err != nil {
		return nil, err
	}

	if info.Identity() == c.hostInfo.Identity() {
		shardItem, err := newHistoryShardsItem(
			shardID,
			c.service,
			c.config,
			c.hostInfo,
			c.historyEventNotifier,
			c.messagePublisher,
			c.domainReplicator,
			c.replicationTaskFetchers,
			c.logger,
			c.throttledLogger,
		)
		if err != nil {
			return nil, err
		}

		c.historyShards[shardID] = shardItem
		c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.ShardItemCreatedCounter)
		shardItem.logger.Info("shard created",
			tag.LifeCycleStarted,
			tag.ComponentShardItem,
			tag.Address(info.Identity()),
			tag.ShardID(shardID),
		)
		return shardItem, nil
	}

	return nil, createShardOwnershipLostError(c.hostInfo.Identity(), info.GetAddress())
}

func (c *shardController) removeHistoryShardItem(
	shardID int,
) (*historyShardsItem, error) {

	nShards := 0
	c.Lock()
	defer c.Unlock()

	shardItem, ok := c.historyShards[shardID]
	if !ok {
		return nil, &shared.InternalServiceError{
			Message: fmt.Sprintf("no item found to remove for shard: %v", shardID),
		}
	}
	delete(c.historyShards, shardID)
	nShards = len(c.historyShards)

	c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.ShardItemRemovedCounter)

	shardItem.logger.Info("shard removed",
		tag.LifeCycleStopped,
		tag.ComponentShardItem,
		tag.Address(c.hostInfo.Identity()),
		tag.ShardID(shardID),
		tag.Number(int64(nShards)),
	)
	return shardItem, nil
}

// shardManagementEventLoop is the main event loop for
// shardController. It is responsible for acquiring /
// releasing shards in response to any event that can
// change the shard ownership. These events are
//   a. Ring membership change
//   b. Periodic membership checking
//   c. ShardOwnershipLostError and subsequent ShardClosedEvents from engine
func (c *shardController) shardManagementEventLoop() {

	defer c.shutdownWG.Done()

	acquireTicker := time.NewTicker(c.config.AcquireShardInterval())
	defer acquireTicker.Stop()

	for {

		select {
		case <-c.shutdownCh:
			c.doShutdown()
			return

		case <-acquireTicker.C:
			c.acquireShards()

		case changedEvent := <-c.membershipUpdateCh:
			c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.MembershipChangedCounter)
			c.logger.Info("encounter member ship change event",
				tag.ValueRingMembershipChangedEvent,
				tag.Address(c.hostInfo.Identity()),
				tag.NumberProcessed(len(changedEvent.HostsAdded)),
				tag.NumberDeleted(len(changedEvent.HostsRemoved)),
				tag.Number(int64(len(changedEvent.HostsUpdated))),
			)
			c.acquireShards()

		case shardID := <-c.shardClosedCh:
			c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.ShardClosedCounter)
			c.logger.Info("encounter shard close event",
				tag.LifeCycleStopping,
				tag.ComponentShard,
				tag.ShardID(shardID),
				tag.Address(c.hostInfo.Identity()),
			)
			c.removeEngineForShard(shardID)
			// The async close notifications can cause a race
			// between acquire/release when nodes are flapping
			// The impact of this race is un-necessary shard load/unloads
			// even though things will settle eventually
			// To reduce the chance of the race happening, lets
			// process all closed events at once before we attempt
			// to acquire new shards again
			c.processShardClosedEvents()
		}
	}
}

func (c *shardController) acquireShards() {

	c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.AcquireShardsCounter)
	sw := c.metricsClient.StartTimer(metrics.HistoryShardControllerScope, metrics.AcquireShardsLatency)
	defer sw.Stop()

AcquireLoop:
	for shardID := 0; shardID < c.config.NumberOfShards; shardID++ {
		info, err := c.historyServiceResolver.Lookup(string(shardID))
		if err != nil {
			c.logger.Error("error looking up host for shardID",
				tag.Error(err),
				tag.OperationFailed,
				tag.ShardID(shardID),
			)
			continue AcquireLoop
		}

		if info.Identity() == c.hostInfo.Identity() {
			_, err := c.getEngineForShard(shardID)
			if err != nil {
				c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.GetEngineForShardErrorCounter)
				c.logger.Error("unable to create history shard engine",
					tag.Error(err),
					tag.OperationFailed,
					tag.ShardID(shardID),
				)
				continue AcquireLoop
			}
		} else {
			c.removeEngineForShard(shardID)
		}
	}

	c.metricsClient.UpdateGauge(metrics.HistoryShardControllerScope, metrics.NumShardsGauge, float64(c.numShards()))
}

func (c *shardController) doShutdown() {
	c.logger.Info("stopping shard controller",
		tag.LifeCycleStopping,
		tag.Address(c.hostInfo.Identity()),
	)
	c.Lock()
	defer c.Unlock()
	for _, item := range c.historyShards {
		item.stopEngine()
	}
	c.historyShards = nil
}

func (c *shardController) processShardClosedEvents() {
	for {
		select {
		case shardID := <-c.shardClosedCh:
			c.metricsClient.IncCounter(metrics.HistoryShardControllerScope, metrics.ShardClosedCounter)
			c.logger.Info("encounter shard close event",
				tag.LifeCycleStopping,
				tag.ComponentShard,
				tag.ShardID(shardID),
				tag.Address(c.hostInfo.Identity()),
			)
			c.removeEngineForShard(shardID)
		default:
			return
		}
	}
}

func (c *shardController) numShards() int {
	nShards := 0
	c.RLock()
	nShards = len(c.historyShards)
	c.RUnlock()
	return nShards
}

func (c *shardController) shardIDs() []int32 {
	c.RLock()
	ids := []int32{}
	for id := range c.historyShards {
		id32 := int32(id)
		ids = append(ids, id32)
	}
	c.RUnlock()
	return ids
}

func (i *historyShardsItem) getOrCreateEngine(
	shardClosedCh chan<- int,
) (Engine, error) {

	i.RLock()
	if i.status == shardsItemStatusStarted {
		defer i.RUnlock()
		return i.engine, nil
	}
	i.RUnlock()

	i.Lock()
	defer i.Unlock()
	switch i.status {
	case shardsItemStatusInitialized:
		i.logger.Info("acquiring shard",
			tag.LifeCycleStarting,
			tag.ComponentShardEngine,
			tag.ShardID(i.shardID),
			tag.Address(i.hostInfo.Identity()),
		)
		context, err := acquireShard(i, shardClosedCh)
		if err != nil {
			return nil, err
		}
		i.engine = context.GetEngine()
		i.engine.Start()
		i.logger.Info("shard acquired",
			tag.LifeCycleStarted,
			tag.ComponentShardEngine,
			tag.ShardID(i.shardID),
			tag.Address(i.hostInfo.Identity()),
		)
		i.status = shardsItemStatusStarted
		return i.engine, nil
	case shardsItemStatusStarted:
		return i.engine, nil
	case shardsItemStatusStopped:
		i.logger.Info("unable to acquire shard",
			tag.LifeCycleStopped,
			tag.ComponentShardEngine,
			tag.ShardID(i.shardID),
			tag.Address(i.hostInfo.Identity()),
		)
		return nil, fmt.Errorf("shard %v for host '%v' is shut down", i.shardID, i.hostInfo.Identity())
	default:
		panic(i.logInvalidStatus())
	}
}

func (i *historyShardsItem) stopEngine() {
	i.Lock()
	defer i.Unlock()

	switch i.status {
	case shardsItemStatusInitialized:
		i.status = shardsItemStatusStopped
	case shardsItemStatusStarted:
		i.logger.Info("stopping shard",
			tag.LifeCycleStopping,
			tag.ComponentShardEngine,
			tag.ShardID(i.shardID),
			tag.Address(i.hostInfo.Identity()),
		)
		i.engine.Stop()
		i.engine = nil
		i.logger.Info("shard stopped",
			tag.LifeCycleStopped,
			tag.ComponentShardEngine,
			tag.ShardID(i.shardID),
			tag.Address(i.hostInfo.Identity()),
		)
		i.status = shardsItemStatusStopped
	case shardsItemStatusStopped:
		// no op
	default:
		panic(i.logInvalidStatus())
	}
}

func (i *historyShardsItem) isValid() bool {
	i.RLock()
	defer i.RUnlock()

	switch i.status {
	case shardsItemStatusInitialized, shardsItemStatusStarted:
		return true
	case shardsItemStatusStopped:
		return false
	default:
		panic(i.logInvalidStatus())
	}
}

func (i *historyShardsItem) logInvalidStatus() string {
	msg := fmt.Sprintf(
		"Host '%v' encounter invalid status %v for shard item for shardID '%v'.",
		i.hostInfo.Identity(), i.status, i.shardID,
	)
	i.logger.Error(msg)
	return msg
}

func isShardOwnershipLostError(err error) bool {
	_, ok := err.(*persistence.ShardOwnershipLostError)
	return ok
}
