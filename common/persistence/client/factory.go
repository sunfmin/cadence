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

package client

import (
	"sync"

	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	"github.com/uber/cadence/common/persistence/cassandra"
	"github.com/uber/cadence/common/persistence/sql"
	"github.com/uber/cadence/common/quotas"
	"github.com/uber/cadence/common/service/config"
)

type (
	// Factory defines the interface for any implementation that can vend
	// persistence layer objects backed by a data store. The actual dat store
	// is implementation detail hidden behind this interface
	Factory interface {
		// Close the factory
		Close()
		// NewTaskManager returns a new task manager
		NewTaskManager() (persistence.TaskManager, error)
		// NewShardManager returns a new shard manager
		NewShardManager() (persistence.ShardManager, error)
		// NewHistoryManager returns a new historyV2 manager
		NewHistoryManager() (persistence.HistoryManager, error)
		// NewMetadataManager returns a new metadata manager
		NewMetadataManager() (persistence.MetadataManager, error)
		// NewExecutionManager returns a new execution manager for a given shardID
		NewExecutionManager(shardID int) (persistence.ExecutionManager, error)
		// NewVisibilityManager returns a new visibility manager
		NewVisibilityManager() (persistence.VisibilityManager, error)
		// NewDomainReplicationQueue returns a new queue for domain replication
		NewDomainReplicationQueue() (persistence.DomainReplicationQueue, error)
	}

	// DataStoreFactory is a low level interface to be implemented by a data store
	// Examples of data stores are cassandra, mysql etc
	DataStoreFactory interface {
		// Close closes the factory
		Close()
		// NewTaskStore returns a new task store
		NewTaskStore() (persistence.TaskStore, error)
		// NewShardStore returns a new shard store
		NewShardStore() (persistence.ShardStore, error)
		// NewHistoryV2Store returns a new historyV2 store
		NewHistoryV2Store() (persistence.HistoryStore, error)
		// NewMetadataStore returns a new metadata store
		NewMetadataStore() (persistence.MetadataStore, error)
		// NewExecutionStore returns an execution store for given shardID
		NewExecutionStore(shardID int) (persistence.ExecutionStore, error)
		// NewVisibilityStore returns a new visibility store
		NewVisibilityStore() (persistence.VisibilityStore, error)
		// NewQueue returns a new queue store
		NewQueue(queueType common.QueueType) (persistence.Queue, error)
	}

	// DataStore represents a data store
	DataStore struct {
		factory     DataStoreFactory
		rateLimiter quotas.Limiter
	}

	factoryImpl struct {
		sync.RWMutex
		config        *config.Persistence
		metricsClient metrics.Client
		logger        log.Logger
		dataStores    map[storeType]DataStore
		clusterName   string
	}

	storeType int
)

const (
	storeTypeHistory storeType = iota + 1
	storeTypeTask
	storeTypeShard
	storeTypeMetadata
	storeTypeExecution
	storeTypeVisibility
	storeTypeQueue
)

var storeTypes = []storeType{
	storeTypeHistory,
	storeTypeTask,
	storeTypeShard,
	storeTypeMetadata,
	storeTypeExecution,
	storeTypeVisibility,
	storeTypeQueue,
}

// New returns an implementation of factory that vends persistence objects based on
// specified configuration. This factory takes as input a config.Persistence object
// which specifies the datastore to be used for a given type of object. This config
// also contains config for individual dataStores themselves.
//
// The objects returned by this factory enforce rateLimiter and maxconns according to
// given configuration. In addition, all objects will emit metrics automatically
func New(
	cfg *config.Persistence,
	clusterName string,
	metricsClient metrics.Client,
	logger log.Logger,
) Factory {
	factory := &factoryImpl{
		config:        cfg,
		metricsClient: metricsClient,
		logger:        logger,
		clusterName:   clusterName,
	}
	limiters := buildRatelimiters(cfg)
	factory.init(clusterName, limiters)
	return factory
}

// NewTaskManager returns a new task manager
func (f *factoryImpl) NewTaskManager() (persistence.TaskManager, error) {
	ds := f.dataStores[storeTypeTask]
	result, err := ds.factory.NewTaskStore()
	if err != nil {
		return nil, err
	}
	if ds.rateLimiter != nil {
		result = persistence.NewTaskPersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewTaskPersistenceMetricsClient(result, f.metricsClient, f.logger)
	}
	return result, nil
}

// NewShardManager returns a new shard manager
func (f *factoryImpl) NewShardManager() (persistence.ShardManager, error) {
	ds := f.dataStores[storeTypeShard]
	result, err := ds.factory.NewShardStore()
	if err != nil {
		return nil, err
	}
	if ds.rateLimiter != nil {
		result = persistence.NewShardPersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewShardPersistenceMetricsClient(result, f.metricsClient, f.logger)
	}
	return result, nil
}

// NewHistoryManager returns a new history manager
func (f *factoryImpl) NewHistoryManager() (persistence.HistoryManager, error) {
	ds := f.dataStores[storeTypeHistory]
	store, err := ds.factory.NewHistoryV2Store()
	if err != nil {
		return nil, err
	}
	result := persistence.NewHistoryV2ManagerImpl(store, f.logger, f.config.TransactionSizeLimit)
	if ds.rateLimiter != nil {
		result = persistence.NewHistoryV2PersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewHistoryV2PersistenceMetricsClient(result, f.metricsClient, f.logger)
	}
	return result, nil
}

// NewMetadataManager returns a new metadata manager
func (f *factoryImpl) NewMetadataManager() (persistence.MetadataManager, error) {
	var err error
	var store persistence.MetadataStore
	ds := f.dataStores[storeTypeMetadata]
	store, err = ds.factory.NewMetadataStore()
	if err != nil {
		return nil, err
	}

	result := persistence.NewMetadataManagerImpl(store, f.logger)
	if ds.rateLimiter != nil {
		result = persistence.NewMetadataPersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewMetadataPersistenceMetricsClient(result, f.metricsClient, f.logger)
	}
	return result, nil
}

// NewExecutionManager returns a new execution manager for a given shardID
func (f *factoryImpl) NewExecutionManager(shardID int) (persistence.ExecutionManager, error) {
	ds := f.dataStores[storeTypeExecution]
	store, err := ds.factory.NewExecutionStore(shardID)
	if err != nil {
		return nil, err
	}
	result := persistence.NewExecutionManagerImpl(store, f.logger)
	if ds.rateLimiter != nil {
		result = persistence.NewWorkflowExecutionPersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewWorkflowExecutionPersistenceMetricsClient(result, f.metricsClient, f.logger)
	}
	return result, nil
}

// NewVisibilityManager returns a new visibility manager
func (f *factoryImpl) NewVisibilityManager() (persistence.VisibilityManager, error) {
	ds := f.dataStores[storeTypeVisibility]
	store, err := ds.factory.NewVisibilityStore()
	if err != nil {
		return nil, err
	}
	visConfig := f.config.VisibilityConfig
	if visConfig != nil && visConfig.EnableReadFromClosedExecutionV2() && f.isCassandra() {
		store, err = cassandra.NewVisibilityPersistenceV2(store, f.getCassandraConfig(), f.logger)
	}

	result := persistence.NewVisibilityManagerImpl(store, f.logger)
	if ds.rateLimiter != nil {
		result = persistence.NewVisibilityPersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if visConfig != nil && visConfig.EnableSampling() {
		result = persistence.NewVisibilitySamplingClient(result, visConfig, f.metricsClient, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewVisibilityPersistenceMetricsClient(result, f.metricsClient, f.logger)
	}

	return result, nil
}

func (f *factoryImpl) NewDomainReplicationQueue() (persistence.DomainReplicationQueue, error) {
	ds := f.dataStores[storeTypeQueue]
	result, err := ds.factory.NewQueue(common.DomainReplicationQueueType)
	if err != nil {
		return nil, err
	}
	if ds.rateLimiter != nil {
		result = persistence.NewQueuePersistenceRateLimitedClient(result, ds.rateLimiter, f.logger)
	}
	if f.metricsClient != nil {
		result = persistence.NewQueuePersistenceMetricsClient(result, f.metricsClient, f.logger)
	}

	return persistence.NewDomainReplicationQueue(result, f.clusterName, f.metricsClient, f.logger), nil
}

// Close closes this factory
func (f *factoryImpl) Close() {
	ds := f.dataStores[storeTypeExecution]
	ds.factory.Close()
}

func (f *factoryImpl) isCassandra() bool {
	cfg := f.config
	return cfg.DataStores[cfg.VisibilityStore].SQL == nil
}

func (f *factoryImpl) getCassandraConfig() *config.Cassandra {
	cfg := f.config
	return cfg.DataStores[cfg.VisibilityStore].Cassandra
}

func (f *factoryImpl) init(clusterName string, limiters map[string]quotas.Limiter) {
	f.dataStores = make(map[storeType]DataStore, len(storeTypes))
	defaultCfg := f.config.DataStores[f.config.DefaultStore]
	defaultDataStore := DataStore{rateLimiter: limiters[f.config.DefaultStore]}
	switch {
	case defaultCfg.Cassandra != nil:
		defaultDataStore.factory = cassandra.NewFactory(*defaultCfg.Cassandra, clusterName, f.logger)
	case defaultCfg.SQL != nil:
		defaultDataStore.factory = sql.NewFactory(*defaultCfg.SQL, clusterName, f.logger)
	default:
		f.logger.Fatal("invalid config: one of cassandra or sql params must be specified")
	}

	for _, st := range storeTypes {
		if st != storeTypeVisibility {
			f.dataStores[st] = defaultDataStore
		}
	}

	visibilityCfg := f.config.DataStores[f.config.VisibilityStore]
	visibilityDataStore := DataStore{rateLimiter: limiters[f.config.VisibilityStore]}
	switch {
	case defaultCfg.Cassandra != nil:
		visibilityDataStore.factory = cassandra.NewFactory(*visibilityCfg.Cassandra, clusterName, f.logger)
	case visibilityCfg.SQL != nil:
		visibilityDataStore.factory = sql.NewFactory(*visibilityCfg.SQL, clusterName, f.logger)
	default:
		f.logger.Fatal("invalid config: one of cassandra or sql params must be specified")
	}

	f.dataStores[storeTypeVisibility] = visibilityDataStore
}

func buildRatelimiters(cfg *config.Persistence) map[string]quotas.Limiter {
	result := make(map[string]quotas.Limiter, len(cfg.DataStores))
	for dsName, ds := range cfg.DataStores {
		qps := 0
		if ds.Cassandra != nil {
			qps = ds.Cassandra.MaxQPS
		}
		if ds.SQL != nil {
			qps = ds.SQL.MaxQPS
		}
		if qps > 0 {
			result[dsName] = quotas.NewSimpleRateLimiter(qps)
		}
	}
	return result
}
