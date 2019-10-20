// Copyright (c) 2019 Uber Technologies, Inc.
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

//go:generate mockgen -copyright_file ../../../LICENSE -package $GOPACKAGE -source $GOFILE -destination Bean_mock.go

package client

import (
	"sync"

	"github.com/uber/cadence/common/persistence"
)

type (
	// Bean in an collection of persistence manager
	Bean interface {
		Close()

		GetMetadataManager() persistence.MetadataManager
		SetMetadataManager(persistence.MetadataManager)

		GetTaskManager() persistence.TaskManager
		SetTaskManager(persistence.TaskManager)

		GetVisibilityManager() persistence.VisibilityManager
		SetVisibilityManager(persistence.VisibilityManager)

		GetDomainReplicationQueue() persistence.DomainReplicationQueue
		SetDomainReplicationQueue(persistence.DomainReplicationQueue)

		GetShardManager() persistence.ShardManager
		SetShardManager(persistence.ShardManager)

		GetHistoryManager() persistence.HistoryManager
		SetHistoryManager(persistence.HistoryManager)

		GetExecutionManager(int) (persistence.ExecutionManager, error)
		SetExecutionManager(int, persistence.ExecutionManager)
	}

	// BeanImpl stores persistence managers
	BeanImpl struct {
		metadataManager         persistence.MetadataManager
		taskManager             persistence.TaskManager
		visibilityManager       persistence.VisibilityManager
		domainReplicationQueue  persistence.DomainReplicationQueue
		shardManager            persistence.ShardManager
		historyManager          persistence.HistoryManager
		executionManagerFactory persistence.ExecutionManagerFactory

		sync.RWMutex
		shardIDToExecutionManager map[int]persistence.ExecutionManager
	}
)

// NewBeanByFactory crate a new store bean using factory
func NewBeanByFactory(
	factory Factory,
) (*BeanImpl, error) {

	metadataMgr, err := factory.NewMetadataManager()
	if err != nil {
		return nil, err
	}

	taskMgr, err := factory.NewTaskManager()
	if err != nil {
		return nil, err
	}

	visibilityMgr, err := factory.NewVisibilityManager()
	if err != nil {
		return nil, err
	}

	domainReplicationQueue, err := factory.NewDomainReplicationQueue()
	if err != nil {
		return nil, err
	}

	shardMgr, err := factory.NewShardManager()
	if err != nil {
		return nil, err
	}

	historyMgr, err := factory.NewHistoryManager()
	if err != nil {
		return nil, err
	}

	return NewBean(
		metadataMgr,
		taskMgr,
		visibilityMgr,
		domainReplicationQueue,
		shardMgr,
		historyMgr,
		factory,
	), nil
}

// NewBean create a new store bean
func NewBean(
	metadataManager persistence.MetadataManager,
	taskManager persistence.TaskManager,
	visibilityManager persistence.VisibilityManager,
	domainReplicationQueue persistence.DomainReplicationQueue,
	shardManager persistence.ShardManager,
	historyManager persistence.HistoryManager,
	executionManagerFactory persistence.ExecutionManagerFactory,
) *BeanImpl {
	return &BeanImpl{
		metadataManager:         metadataManager,
		taskManager:             taskManager,
		visibilityManager:       visibilityManager,
		domainReplicationQueue:  domainReplicationQueue,
		shardManager:            shardManager,
		historyManager:          historyManager,
		executionManagerFactory: executionManagerFactory,

		shardIDToExecutionManager: make(map[int]persistence.ExecutionManager),
	}
}

// GetMetadataManager get MetadataManager
func (s *BeanImpl) GetMetadataManager() persistence.MetadataManager {
	return s.metadataManager
}

// SetMetadataManager set MetadataManager
func (s *BeanImpl) SetMetadataManager(
	metadataManager persistence.MetadataManager,
) {

	s.metadataManager = metadataManager
}

// GetTaskManager get TaskManager
func (s *BeanImpl) GetTaskManager() persistence.TaskManager {
	return s.taskManager
}

// SetTaskManager set TaskManager
func (s *BeanImpl) SetTaskManager(
	taskManager persistence.TaskManager,
) {

	s.taskManager = taskManager
}

// GetVisibilityManager get VisibilityManager
func (s *BeanImpl) GetVisibilityManager() persistence.VisibilityManager {
	return s.visibilityManager
}

// SetVisibilityManager set VisibilityManager
func (s *BeanImpl) SetVisibilityManager(
	visibilityManager persistence.VisibilityManager,
) {

	s.visibilityManager = visibilityManager
}

// GetDomainReplicationQueue get DomainReplicationQueue
func (s *BeanImpl) GetDomainReplicationQueue() persistence.DomainReplicationQueue {
	return s.domainReplicationQueue
}

// SetDomainReplicationQueue set DomainReplicationQueue
func (s *BeanImpl) SetDomainReplicationQueue(
	domainReplicationQueue persistence.DomainReplicationQueue,
) {

	s.domainReplicationQueue = domainReplicationQueue
}

// GetShardManager get ShardManager
func (s *BeanImpl) GetShardManager() persistence.ShardManager {
	return s.shardManager
}

// SetShardManager set ShardManager
func (s *BeanImpl) SetShardManager(
	shardManager persistence.ShardManager,
) {

	s.shardManager = shardManager
}

// GetHistoryManager get HistoryManager
func (s *BeanImpl) GetHistoryManager() persistence.HistoryManager {
	return s.historyManager
}

// SetHistoryManager set HistoryManager
func (s *BeanImpl) SetHistoryManager(
	historyManager persistence.HistoryManager,
) {

	s.historyManager = historyManager
}

// GetExecutionManager get ExecutionManager
func (s *BeanImpl) GetExecutionManager(
	shardID int,
) (persistence.ExecutionManager, error) {

	s.RLock()
	executionManager, ok := s.shardIDToExecutionManager[shardID]
	if ok {
		s.RUnlock()
		return executionManager, nil
	}
	s.RUnlock()

	s.Lock()
	defer s.Unlock()

	executionManager, ok = s.shardIDToExecutionManager[shardID]
	if ok {
		return executionManager, nil
	}

	executionManager, err := s.executionManagerFactory.NewExecutionManager(shardID)
	if err != nil {
		return nil, err
	}

	s.shardIDToExecutionManager[shardID] = executionManager
	return executionManager, nil
}

// SetExecutionManager set ExecutionManager
func (s *BeanImpl) SetExecutionManager(
	shardID int,
	executionManager persistence.ExecutionManager,
) {

	s.Lock()
	defer s.Unlock()
	s.shardIDToExecutionManager[shardID] = executionManager
}

// Close cleanup connections
func (s *BeanImpl) Close() {
	s.Lock()
	defer s.Unlock()

	s.metadataManager.Close()
	s.taskManager.Close()
	s.visibilityManager.Close()
	s.domainReplicationQueue.Close()
	s.shardManager.Close()
	s.historyManager.Close()
	s.executionManagerFactory.Close()
	for _, executionMgr := range s.shardIDToExecutionManager {
		executionMgr.Close()
	}
}
