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
	"math/rand"
	"os"
	"sync/atomic"
	"time"

	"github.com/uber-go/tally"
	"go.uber.org/cadence/.gen/go/cadence/workflowserviceclient"
	"go.uber.org/yarpc"

	"github.com/uber/cadence/client"
	"github.com/uber/cadence/client/admin"
	"github.com/uber/cadence/client/frontend"
	"github.com/uber/cadence/client/history"
	"github.com/uber/cadence/client/matching"
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/archiver"
	"github.com/uber/cadence/common/archiver/provider"
	"github.com/uber/cadence/common/cache"
	"github.com/uber/cadence/common/clock"
	"github.com/uber/cadence/common/cluster"
	"github.com/uber/cadence/common/elasticsearch"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/loggerimpl"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/membership"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/metrics"
	"github.com/uber/cadence/common/persistence"
	persistenceClient "github.com/uber/cadence/common/persistence/client"
	"github.com/uber/cadence/common/service/config"
	"github.com/uber/cadence/common/service/dynamicconfig"
)

var cadenceServices = []string{
	common.FrontendServiceName,
	common.HistoryServiceName,
	common.MatchingServiceName,
	common.WorkerServiceName,
}

type (
	// BootstrapParams holds the set of parameters
	// needed to bootstrap a service
	BootstrapParams struct {
		Name       string
		InstanceID string
		Logger     log.Logger

		MetricScope         tally.Scope
		MembershipFactory   MembershipMonitorFactory
		RPCFactory          common.RPCFactory
		PProfInitializer    common.PProfInitializer
		PersistenceConfig   config.Persistence
		ClusterMetadata     cluster.Metadata
		ReplicatorConfig    config.Replicator
		MetricsClient       metrics.Client
		MessagingClient     messaging.Client
		ESClient            elasticsearch.Client
		ESConfig            *elasticsearch.Config
		DynamicConfig       dynamicconfig.Client
		DispatcherProvider  client.DispatcherProvider
		DCRedirectionPolicy config.DCRedirectionPolicy
		PublicClient        workflowserviceclient.Interface
		ArchivalMetadata    archiver.ArchivalMetadata
		ArchiverProvider    provider.ArchiverProvider
	}

	// MembershipMonitorFactory provides a bootstrapped membership monitor
	MembershipMonitorFactory interface {
		// Create vends a bootstrapped membership monitor
		Create(d *yarpc.Dispatcher) (membership.Monitor, error)
	}

	// VisibilityManagerInitializer is the function each service should implement
	// for visibility manager initialization
	VisibilityManagerInitializer func(
		persistenceBean persistenceClient.Bean,
		logger log.Logger,
	) (persistence.VisibilityManager, error)

	// Service contains the objects specific to this service
	serviceImpl struct {
		status int32

		// static infos

		numShards       int
		serviceName     string
		hostName        string
		metricsScope    tally.Scope
		clusterMetadata cluster.Metadata

		// other common resources

		domainCache       cache.DomainCache
		timeSource        clock.TimeSource
		payloadSerializer persistence.PayloadSerializer
		metricsClient     metrics.Client
		messagingClient   messaging.Client
		archivalMetadata  archiver.ArchivalMetadata
		archiverProvider  provider.ArchiverProvider

		// membership infos

		membershipMonitor       membership.Monitor
		frontendServiceResolver membership.ServiceResolver
		matchingServiceResolver membership.ServiceResolver
		historyServiceResolver  membership.ServiceResolver
		workerServiceResolver   membership.ServiceResolver

		// internal services clients

		publicClient      workflowserviceclient.Interface
		frontendRawClient frontend.Client
		frontendClient    frontend.Client
		matchingRawClient matching.Client
		matchingClient    matching.Client
		historyRawClient  history.Client
		historyClient     history.Client
		clientBean        client.Bean

		// persistence clients

		persistenceBean persistenceClient.Bean
		visibilityMgr   persistence.VisibilityManager

		// loggers

		logger          log.Logger
		throttledLogger log.Logger

		// for registering handlers
		dispatcher *yarpc.Dispatcher

		// internal vars

		pprofInitializer       common.PProfInitializer
		runtimeMetricsReporter *metrics.RuntimeMetricsReporter
		membershipFactory      MembershipMonitorFactory
		rpcFactory             common.RPCFactory
	}
)

var _ Service = (*serviceImpl)(nil)

// NewService create a new server containing common dependencies
func NewService(
	params *BootstrapParams,
	serviceName string,
	throttledLoggerMaxRPS dynamicconfig.IntPropertyFn,
	visibilityManagerInitializer VisibilityManagerInitializer,
) (Service, error) {

	logger := params.Logger.WithTags(tag.Service(serviceName))
	throttledLogger := loggerimpl.NewThrottledLogger(logger, throttledLoggerMaxRPS)

	numShards := params.PersistenceConfig.NumHistoryShards
	hostName, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	dynamicCollection := dynamicconfig.NewCollection(params.DynamicConfig, logger)

	dispatcher := params.RPCFactory.CreateDispatcher()
	membershipMonitor, err := params.MembershipFactory.Create(dispatcher)
	if err != nil {
		return nil, err
	}

	clientBean, err := client.NewClientBean(
		client.NewRPCClientFactory(
			params.RPCFactory,
			membershipMonitor,
			params.MetricsClient,
			dynamicCollection,
			numShards,
			logger,
		),
		params.DispatcherProvider,
		params.ClusterMetadata,
	)
	if err != nil {
		return nil, err
	}

	persistenceBean, err := persistenceClient.NewBeanByFactory(persistenceClient.New(
		&params.PersistenceConfig,
		params.ClusterMetadata.GetCurrentClusterName(),
		params.MetricsClient,
		logger,
	))
	if err != nil {
		return nil, err
	}
	visibilityMgr, err := visibilityManagerInitializer(
		persistenceBean,
		logger,
	)
	if err != nil {
		return nil, err
	}

	frontendServiceResolver, err := membershipMonitor.GetResolver(common.FrontendServiceName)
	if err != nil {
		return nil, err
	}

	matchingServiceResolver, err := membershipMonitor.GetResolver(common.MatchingServiceName)
	if err != nil {
		return nil, err
	}

	historyServiceResolver, err := membershipMonitor.GetResolver(common.HistoryServiceName)
	if err != nil {
		return nil, err
	}

	workerServiceResolver, err := membershipMonitor.GetResolver(common.WorkerServiceName)
	if err != nil {
		return nil, err
	}

	domainCache := cache.NewDomainCache(
		persistenceBean.GetMetadataManager(),
		params.ClusterMetadata,
		params.MetricsClient,
		logger,
	)

	frontendRawClient := clientBean.GetFrontendClient()
	frontendClient := frontend.NewRetryableClient(
		frontendRawClient,
		common.CreateFrontendServiceRetryPolicy(),
		common.IsWhitelistServiceTransientError,
	)

	matchingRawClient, err := clientBean.GetMatchingClient(domainCache.GetDomainName)
	if err != nil {
		return nil, err
	}
	matchingClient := matching.NewRetryableClient(
		matchingRawClient,
		common.CreateMatchingServiceRetryPolicy(),
		common.IsWhitelistServiceTransientError,
	)

	historyRawClient := clientBean.GetHistoryClient()
	historyClient := history.NewRetryableClient(
		historyRawClient,
		common.CreateHistoryServiceRetryPolicy(),
		common.IsWhitelistServiceTransientError,
	)

	return &serviceImpl{
		status: common.DaemonStatusInitialized,

		// static infos

		numShards:       numShards,
		serviceName:     params.Name,
		hostName:        hostName,
		metricsScope:    params.MetricScope,
		clusterMetadata: params.ClusterMetadata,

		// other common resources

		domainCache:       domainCache,
		timeSource:        clock.NewRealTimeSource(),
		payloadSerializer: persistence.NewPayloadSerializer(),
		metricsClient:     params.MetricsClient,
		messagingClient:   params.MessagingClient,
		archivalMetadata:  params.ArchivalMetadata,
		archiverProvider:  params.ArchiverProvider,

		// membership infos

		membershipMonitor:       membershipMonitor,
		frontendServiceResolver: frontendServiceResolver,
		matchingServiceResolver: matchingServiceResolver,
		historyServiceResolver:  historyServiceResolver,
		workerServiceResolver:   workerServiceResolver,

		// internal services clients

		publicClient:      params.PublicClient,
		frontendRawClient: frontendRawClient,
		frontendClient:    frontendClient,
		matchingRawClient: matchingRawClient,
		matchingClient:    matchingClient,
		historyRawClient:  historyRawClient,
		historyClient:     historyClient,
		clientBean:        clientBean,

		// persistence clients

		persistenceBean: persistenceBean,
		visibilityMgr:   visibilityMgr,

		// loggers

		logger:          logger,
		throttledLogger: throttledLogger,

		// for registering handlers
		dispatcher: dispatcher,

		// internal vars
		pprofInitializer: params.PProfInitializer,
		runtimeMetricsReporter: metrics.NewRuntimeMetricsReporter(
			params.MetricScope,
			time.Minute,
			logger,
			params.InstanceID,
		),
		membershipFactory: params.MembershipFactory,
		rpcFactory:        params.RPCFactory,
	}, nil
}

func (h *serviceImpl) Start() {

	if !atomic.CompareAndSwapInt32(
		&h.status,
		common.DaemonStatusInitialized,
		common.DaemonStatusStarted,
	) {
		return
	}

	h.metricsScope.Counter(metrics.RestartCount).Inc(1)
	h.runtimeMetricsReporter.Start()

	if err := h.pprofInitializer.Start(); err != nil {
		h.logger.WithTags(tag.Error(err)).Fatal("fail to start PProf")
	}

	if err := h.dispatcher.Start(); err != nil {
		h.logger.WithTags(tag.Error(err)).Fatal("fail to start yarpc dispatcher")
	}

	if err := h.membershipMonitor.Start(); err != nil {
		h.logger.WithTags(tag.Error(err)).Fatal("fail to start membership monitor")
	}

	h.domainCache.Start()

	// The service is now started up
	h.logger.Info("service started")
	// seed the random generator once for this service
	rand.Seed(time.Now().UTC().UnixNano())
}

func (h *serviceImpl) Stop() {

	if !atomic.CompareAndSwapInt32(
		&h.status,
		common.DaemonStatusStarted,
		common.DaemonStatusStopped,
	) {
		return
	}

	h.domainCache.Stop()
	h.membershipMonitor.Stop()
	if err := h.dispatcher.Stop(); err != nil {
		h.logger.WithTags(tag.Error(err)).Error("failed to stop dispatcher")
	}
	h.runtimeMetricsReporter.Stop()
	h.persistenceBean.Close()
}

func (h *serviceImpl) GetServiceName() string {
	return h.serviceName
}

func (h *serviceImpl) GetHostName() string {
	return h.hostName
}

func (h *serviceImpl) GetHostInfo() (*membership.HostInfo, error) {
	return h.membershipMonitor.WhoAmI()
}

func (h *serviceImpl) GetClusterMetadata() cluster.Metadata {
	return h.clusterMetadata
}

// other common resources

func (h *serviceImpl) GetDomainCache() cache.DomainCache {
	return h.domainCache
}

func (h *serviceImpl) GetTimeSource() clock.TimeSource {
	return h.timeSource
}

func (h *serviceImpl) GetPayloadSerializer() persistence.PayloadSerializer {
	return h.payloadSerializer
}

func (h *serviceImpl) GetMetricsClient() metrics.Client {
	return h.metricsClient
}

func (h *serviceImpl) GetMessagingClient() messaging.Client {
	return h.messagingClient
}

func (h *serviceImpl) GetArchivalMetadata() archiver.ArchivalMetadata {
	return h.archivalMetadata
}

func (h *serviceImpl) GetArchiverProvider() provider.ArchiverProvider {
	return h.archiverProvider
}

// membership infos

func (h *serviceImpl) GetMembershipMonitor() membership.Monitor {
	return h.membershipMonitor
}

func (h *serviceImpl) GetFrontendServiceResolver() membership.ServiceResolver {
	return h.historyServiceResolver
}

func (h *serviceImpl) GetMatchingServiceResolver() membership.ServiceResolver {
	return h.matchingServiceResolver
}

func (h *serviceImpl) GetHistoryServiceResolver() membership.ServiceResolver {
	return h.historyServiceResolver
}

func (h *serviceImpl) GetWorkerServiceResolver() membership.ServiceResolver {
	return h.workerServiceResolver
}

// internal services clients

func (h *serviceImpl) GetPublicClient() workflowserviceclient.Interface {
	return h.publicClient
}

func (h *serviceImpl) GetFrontendRawClient() frontend.Client {
	return h.frontendRawClient
}

func (h *serviceImpl) GetFrontendClient() frontend.Client {
	return h.frontendClient
}

func (h *serviceImpl) GetMatchingRawClient() matching.Client {
	return h.matchingRawClient
}

func (h *serviceImpl) GetMatchingClient() matching.Client {
	return h.matchingClient
}

func (h *serviceImpl) GetHistoryRawClient() history.Client {
	return h.historyRawClient
}

func (h *serviceImpl) GetHistoryClient() history.Client {
	return h.historyClient
}

func (h *serviceImpl) GetRemoteAdminClient(
	cluster string,
) admin.Client {

	return h.clientBean.GetRemoteAdminClient(cluster)
}

func (h *serviceImpl) GetRemoteFrontendClient(
	cluster string,
) frontend.Client {

	return h.clientBean.GetRemoteFrontendClient(cluster)
}

func (h *serviceImpl) GetClientBean() client.Bean {
	return h.clientBean
}

// persistence clients

func (h *serviceImpl) GetMetadataManager() persistence.MetadataManager {
	return h.persistenceBean.GetMetadataManager()
}

func (h *serviceImpl) GetTaskManager() persistence.TaskManager {
	return h.persistenceBean.GetTaskManager()
}

func (h *serviceImpl) GetVisibilityManager() persistence.VisibilityManager {
	return h.visibilityMgr
}

func (h *serviceImpl) GetDomainReplicationQueue() persistence.DomainReplicationQueue {
	return h.persistenceBean.GetDomainReplicationQueue()
}

func (h *serviceImpl) GetShardManager() persistence.ShardManager {
	return h.persistenceBean.GetShardManager()
}

func (h *serviceImpl) GetHistoryManager() persistence.HistoryManager {
	return h.persistenceBean.GetHistoryManager()
}

func (h *serviceImpl) GetExecutionManager(
	shardID int,
) (persistence.ExecutionManager, error) {

	return h.persistenceBean.GetExecutionManager(shardID)
}

func (h *serviceImpl) GetPersistenceBean() persistenceClient.Bean {
	return h.persistenceBean
}

// loggers

func (h *serviceImpl) GetLogger() log.Logger {
	return h.logger
}

func (h *serviceImpl) GetThrottledLogger() log.Logger {
	return h.throttledLogger
}

// for registering handlers
func (h *serviceImpl) GetDispatcher() *yarpc.Dispatcher {
	return h.dispatcher
}

// GetMetricsServiceIdx returns the metrics name
func GetMetricsServiceIdx(serviceName string, logger log.Logger) metrics.ServiceIdx {
	switch serviceName {
	case common.FrontendServiceName:
		return metrics.Frontend
	case common.HistoryServiceName:
		return metrics.History
	case common.MatchingServiceName:
		return metrics.Matching
	case common.WorkerServiceName:
		return metrics.Worker
	default:
		logger.Fatal("Unknown service name '%v' for metrics!", tag.Service(serviceName))
	}

	// this should never happen!
	return metrics.NumServices
}
