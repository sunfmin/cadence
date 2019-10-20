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

package service

import (
	"github.com/golang/mock/gomock"
	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	publicservicetest "go.uber.org/cadence/.gen/go/cadence/workflowservicetest"

	"github.com/uber/cadence/.gen/go/cadence/workflowservicetest"
	"github.com/uber/cadence/.gen/go/history/historyservicetest"
	"github.com/uber/cadence/.gen/go/matching/matchingservicetest"
	"github.com/uber/cadence/client"
	"github.com/uber/cadence/client/admin"
	"github.com/uber/cadence/client/frontend"
	"github.com/uber/cadence/client/history"
	"github.com/uber/cadence/client/matching"
	"github.com/uber/cadence/common/archiver"
	"github.com/uber/cadence/common/archiver/provider"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/membership"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	persistenceClient "github.com/uber/cadence/common/persistence/client"

	"go.uber.org/yarpc"
	"go.uber.org/zap"
)

type (
	// serviceTest is the test implementation used for testing
	serviceTest struct {
		metricsScope    tally.Scope
		clusterMetadata *cluster.MockMetadata

		// other common resources

		domainCache       *cache.MockDomainCache
		timeSource        clock.TimeSource
		payloadSerializer persistence.PayloadSerializer
		metricsClient     metrics.Client
		messagingClient   *mocks.MessagingClient
		archivalMetadata  *archiver.MockArchivalMetadata
		archiverProvider  *provider.MockArchiverProvider

		// membership infos

		membershipMonitor       membership.Monitor
		frontendServiceResolver membership.ServiceResolver
		matchingServiceResolver membership.ServiceResolver
		historyServiceResolver  membership.ServiceResolver
		workerServiceResolver   membership.ServiceResolver

		// internal services clients

		publicClient   *publicservicetest.MockClient
		frontendClient *workflowservicetest.MockClient
		matchingClient *matchingservicetest.MockClient
		historyClient  *historyservicetest.MockClient
		clientBean     *client.MockBean

		// persistence clients

		persistenceBean *persistenceClient.MockBean
		visibilityMgr   *mocks.VisibilityManager

		logger log.Logger
	}
)

var _ Service = (*serviceTest)(nil)

const (
	testHostName = "test_host"
)

var (
	testHostInfo = membership.NewHostInfo(testHostName, nil)
)

// NewTestService is the new service instance created for testing
func NewTestService(
	controller *gomock.Controller,
) Service {

	zapLogger, err := zap.NewDevelopment()
	if err != nil {
		panic(err)
	}
	logger := loggerimpl.NewLogger(zapLogger)

	return &serviceTest{
		metricsScope:    tally.NoopScope,
		clusterMetadata: cluster.NewMockMetadata(controller),

		// other common resources

		domainCache:       cache.NewMockDomainCache(controller),
		timeSource:        clock.NewRealTimeSource(),
		payloadSerializer: persistence.NewPayloadSerializer(),
		metricsClient:     metrics.NewClient(tally.NoopScope, metrics.Common),
		messagingClient:   &mocks.MessagingClient{},
		archivalMetadata:  &archiver.MockArchivalMetadata{},
		archiverProvider:  &provider.MockArchiverProvider{},

		// membership infos

		membershipMonitor:       nil,
		frontendServiceResolver: nil,
		matchingServiceResolver: nil,
		historyServiceResolver:  nil,
		workerServiceResolver:   nil,

		// internal services clients

		publicClient:   publicservicetest.NewMockClient(controller),
		frontendClient: workflowservicetest.NewMockClient(controller),
		matchingClient: matchingservicetest.NewMockClient(controller),
		historyClient:  historyservicetest.NewMockClient(controller),
		clientBean:     client.NewMockBean(controller),

		// persistence clients

		persistenceBean: persistenceClient.NewMockBean(controller),
		visibilityMgr:   &mocks.VisibilityManager{},

		// logger

		logger: logger,
	}
}

func (s *serviceTest) Start() {

}

func (s *serviceTest) Stop() {

}

// static infos

func (s *serviceTest) GetServiceName() string {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetHostName() string {
	return s.hostInfo.GetAddress()
}

func (s *serviceTest) GetHostInfo() (*membership.HostInfo, error) {
	return s.hostInfo, nil
}

func (s *serviceTest) GetClusterMetadata() cluster.Metadata {
	return s.clusterMetadata
}

// other common resources

func (s *serviceTest) GetDomainCache() cache.DomainCache {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetTimeSource() clock.TimeSource {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetPayloadSerializer() persistence.PayloadSerializer {
	return s.serializer
}

func (s *serviceTest) GetMetricsClient() metrics.Client {
	return s.metricsClient
}

func (s *serviceTest) GetMessagingClient() messaging.Client {
	return s.messagingClient
}

func (s *serviceTest) GetArchivalMetadata() archiver.ArchivalMetadata {
	return s.archivalMetadata
}

func (s *serviceTest) GetArchiverProvider() provider.ArchiverProvider {
	return s.archiverProvider
}

// membership infos

func (s *serviceTest) GetMembershipMonitor() membership.Monitor {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetFrontendServiceResolver() membership.ServiceResolver {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetMatchingServiceResolver() membership.ServiceResolver {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetHistoryServiceResolver() membership.ServiceResolver {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetWorkerServiceResolver() membership.ServiceResolver {
	panic("user should implement this method for test")
}

// internal services clients

func (s *serviceTest) GetPublicClient() workflowserviceclient.Interface {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetFrontendRawClient() frontend.Client {
	return s.clientBean.GetFrontendClient()
}

func (s *serviceTest) GetFrontendClient() frontend.Client {
	return s.clientBean.GetFrontendClient()
}

func (s *serviceTest) GetMatchingRawClient() matching.Client {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetMatchingClient() matching.Client {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetHistoryRawClient() history.Client {
	return s.clientBean.GetHistoryClient()
}

func (s *serviceTest) GetHistoryClient() history.Client {
	return s.clientBean.GetHistoryClient()
}

func (s *serviceTest) GetRemoteAdminClient(
	cluster string,
) admin.Client {

	return s.clientBean.GetRemoteAdminClient(cluster)
}

func (s *serviceTest) GetRemoteFrontendClient(
	cluster string,
) frontend.Client {

	return s.clientBean.GetRemoteFrontendClient(cluster)
}

func (s *serviceTest) GetClientBean() client.Bean {
	return s.clientBean
}

// persistence clients

func (s *serviceTest) GetMetadataManager() persistence.MetadataManager {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetTaskManager() persistence.TaskManager {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetVisibilityManager() persistence.VisibilityManager {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetDomainReplicationQueue() persistence.DomainReplicationQueue {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetShardManager() persistence.ShardManager {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetHistoryManager() persistence.HistoryManager {
	panic("user should implement this method for test")
}

func (s *serviceTest) GetExecutionManager(
	shardID int,
) (persistence.ExecutionManager, error) {

	panic("user should implement this method for test")
}

func (s *serviceTest) GetPersistenceBean() persistenceClient.Bean {
	panic("user should implement this method for test")
}

// loggers

func (s *serviceTest) GetLogger() log.Logger {
	return s.logger
}

func (s *serviceTest) GetThrottledLogger() log.Logger {
	return s.logger
}

// for registering handlers
func (s *serviceTest) GetDispatcher() *yarpc.Dispatcher {
	panic("user should implement this method for test")
}
