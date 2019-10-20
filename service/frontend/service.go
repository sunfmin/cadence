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

package frontend

import (
	"github.com/uber/cadence/common"
	"github.com/uber/cadence/common/archiver"
	"github.com/uber/cadence/common/definition"
	"github.com/uber/cadence/common/domain"
	"github.com/uber/cadence/common/log"
	"github.com/uber/cadence/common/log/tag"
	"github.com/uber/cadence/common/messaging"
	"github.com/uber/cadence/common/mocks"
	"github.com/uber/cadence/common/persistence"
	persistenceClient "github.com/uber/cadence/common/persistence/client"
	persistenceElasticSearch "github.com/uber/cadence/common/persistence/elasticsearch"
	"github.com/uber/cadence/common/service"
	"github.com/uber/cadence/common/service/config"
	"github.com/uber/cadence/common/service/dynamicconfig"
)

// Config represents configuration for cadence-frontend service
type Config struct {
	NumHistoryShards                int
	PersistenceMaxQPS               dynamicconfig.IntPropertyFn
	VisibilityMaxPageSize           dynamicconfig.IntPropertyFnWithDomainFilter
	EnableVisibilitySampling        dynamicconfig.BoolPropertyFn
	EnableReadFromClosedExecutionV2 dynamicconfig.BoolPropertyFn
	VisibilityListMaxQPS            dynamicconfig.IntPropertyFnWithDomainFilter
	EnableReadVisibilityFromES      dynamicconfig.BoolPropertyFnWithDomainFilter
	ESVisibilityListMaxQPS          dynamicconfig.IntPropertyFnWithDomainFilter
	ESIndexMaxResultWindow          dynamicconfig.IntPropertyFn
	HistoryMaxPageSize              dynamicconfig.IntPropertyFnWithDomainFilter
	RPS                             dynamicconfig.IntPropertyFn
	DomainRPS                       dynamicconfig.IntPropertyFnWithDomainFilter
	MaxIDLengthLimit                dynamicconfig.IntPropertyFn
	EnableClientVersionCheck        dynamicconfig.BoolPropertyFn
	MinRetentionDays                dynamicconfig.IntPropertyFn

	// Persistence settings
	HistoryMgrNumConns dynamicconfig.IntPropertyFn

	MaxBadBinaries dynamicconfig.IntPropertyFnWithDomainFilter

	// security protection settings
	EnableAdminProtection         dynamicconfig.BoolPropertyFn
	AdminOperationToken           dynamicconfig.StringPropertyFn
	DisableListVisibilityByFilter dynamicconfig.BoolPropertyFnWithDomainFilter

	// size limit system protection
	BlobSizeLimitError dynamicconfig.IntPropertyFnWithDomainFilter
	BlobSizeLimitWarn  dynamicconfig.IntPropertyFnWithDomainFilter

	ThrottledLogRPS dynamicconfig.IntPropertyFn

	// Domain specific config
	EnableDomainNotActiveAutoForwarding dynamicconfig.BoolPropertyFnWithDomainFilter

	// ValidSearchAttributes is legal indexed keys that can be used in list APIs
	ValidSearchAttributes             dynamicconfig.MapPropertyFn
	SearchAttributesNumberOfKeysLimit dynamicconfig.IntPropertyFnWithDomainFilter
	SearchAttributesSizeOfValueLimit  dynamicconfig.IntPropertyFnWithDomainFilter
	SearchAttributesTotalSizeLimit    dynamicconfig.IntPropertyFnWithDomainFilter
}

// NewConfig returns new service config with default values
func NewConfig(dc *dynamicconfig.Collection, numHistoryShards int, enableReadFromES bool) *Config {
	return &Config{
		NumHistoryShards:                    numHistoryShards,
		PersistenceMaxQPS:                   dc.GetIntProperty(dynamicconfig.FrontendPersistenceMaxQPS, 2000),
		VisibilityMaxPageSize:               dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendVisibilityMaxPageSize, 1000),
		EnableVisibilitySampling:            dc.GetBoolProperty(dynamicconfig.EnableVisibilitySampling, true),
		EnableReadFromClosedExecutionV2:     dc.GetBoolProperty(dynamicconfig.EnableReadFromClosedExecutionV2, false),
		VisibilityListMaxQPS:                dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendVisibilityListMaxQPS, 1),
		EnableReadVisibilityFromES:          dc.GetBoolPropertyFnWithDomainFilter(dynamicconfig.EnableReadVisibilityFromES, enableReadFromES),
		ESVisibilityListMaxQPS:              dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendESVisibilityListMaxQPS, 3),
		ESIndexMaxResultWindow:              dc.GetIntProperty(dynamicconfig.FrontendESIndexMaxResultWindow, 10000),
		HistoryMaxPageSize:                  dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendHistoryMaxPageSize, common.GetHistoryMaxPageSize),
		RPS:                                 dc.GetIntProperty(dynamicconfig.FrontendRPS, 1200),
		DomainRPS:                           dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendDomainRPS, 1200),
		MaxIDLengthLimit:                    dc.GetIntProperty(dynamicconfig.MaxIDLengthLimit, 1000),
		HistoryMgrNumConns:                  dc.GetIntProperty(dynamicconfig.FrontendHistoryMgrNumConns, 10),
		MaxBadBinaries:                      dc.GetIntPropertyFilteredByDomain(dynamicconfig.FrontendMaxBadBinaries, domain.MaxBadBinaries),
		EnableAdminProtection:               dc.GetBoolProperty(dynamicconfig.EnableAdminProtection, false),
		AdminOperationToken:                 dc.GetStringProperty(dynamicconfig.AdminOperationToken, common.DefaultAdminOperationToken),
		DisableListVisibilityByFilter:       dc.GetBoolPropertyFnWithDomainFilter(dynamicconfig.DisableListVisibilityByFilter, false),
		BlobSizeLimitError:                  dc.GetIntPropertyFilteredByDomain(dynamicconfig.BlobSizeLimitError, 2*1024*1024),
		BlobSizeLimitWarn:                   dc.GetIntPropertyFilteredByDomain(dynamicconfig.BlobSizeLimitWarn, 256*1024),
		ThrottledLogRPS:                     dc.GetIntProperty(dynamicconfig.FrontendThrottledLogRPS, 20),
		EnableDomainNotActiveAutoForwarding: dc.GetBoolPropertyFnWithDomainFilter(dynamicconfig.EnableDomainNotActiveAutoForwarding, false),
		EnableClientVersionCheck:            dc.GetBoolProperty(dynamicconfig.EnableClientVersionCheck, false),
		ValidSearchAttributes:               dc.GetMapProperty(dynamicconfig.ValidSearchAttributes, definition.GetDefaultIndexedKeys()),
		SearchAttributesNumberOfKeysLimit:   dc.GetIntPropertyFilteredByDomain(dynamicconfig.SearchAttributesNumberOfKeysLimit, 100),
		SearchAttributesSizeOfValueLimit:    dc.GetIntPropertyFilteredByDomain(dynamicconfig.SearchAttributesSizeOfValueLimit, 2*1024),
		SearchAttributesTotalSizeLimit:      dc.GetIntPropertyFilteredByDomain(dynamicconfig.SearchAttributesTotalSizeLimit, 40*1024),
		MinRetentionDays:                    dc.GetIntProperty(dynamicconfig.MinRetentionDays, domain.MinRetentionDays),
	}
}

// Service represents the cadence-frontend service
type Service struct {
	service.Service
	config *Config

	numHistoryShards int
	bootstrapParams  *service.BootstrapParams

	stopC chan struct{}
}

// NewService builds a new cadence-frontend service
func NewService(
	params *service.BootstrapParams,
) (service.Service, error) {

	serviceConfig := NewConfig(
		dynamicconfig.NewCollection(params.DynamicConfig, params.Logger),
		params.PersistenceConfig.NumHistoryShards,
		len(params.PersistenceConfig.AdvancedVisibilityStore) != 0,
	)

	params.PersistenceConfig.SetMaxQPS(params.PersistenceConfig.DefaultStore, serviceConfig.PersistenceMaxQPS())
	params.PersistenceConfig.HistoryMaxConns = serviceConfig.HistoryMgrNumConns()
	params.PersistenceConfig.VisibilityConfig = &config.VisibilityConfig{
		VisibilityListMaxQPS:            serviceConfig.VisibilityListMaxQPS,
		EnableSampling:                  serviceConfig.EnableVisibilitySampling,
		EnableReadFromClosedExecutionV2: serviceConfig.EnableReadFromClosedExecutionV2,
	}

	baseService, err := service.NewService(
		params,
		common.FrontendServiceName,
		serviceConfig.ThrottledLogRPS,
		func(
			persistenceBean persistenceClient.Bean,
			logger log.Logger,
		) (persistence.VisibilityManager, error) {

			if params.ESConfig == nil {
				return persistence.NewVisibilityManagerWrapper(
					persistenceBean.GetVisibilityManager(),
					nil,
					serviceConfig.EnableReadVisibilityFromES,
					// frontend visibility never write
					dynamicconfig.GetStringPropertyFn(common.AdvancedVisibilityWritingModeOff),
				), nil
			}

			return persistence.NewVisibilityManagerWrapper(
				persistenceBean.GetVisibilityManager(),
				persistenceElasticSearch.NewESVisibilityManager(
					params.ESConfig.Indices[common.VisibilityAppName],
					params.ESClient,
					&config.VisibilityConfig{
						MaxQPS:                 serviceConfig.PersistenceMaxQPS,
						VisibilityListMaxQPS:   serviceConfig.ESVisibilityListMaxQPS,
						ESIndexMaxResultWindow: serviceConfig.ESIndexMaxResultWindow,
						ValidSearchAttributes:  serviceConfig.ValidSearchAttributes,
					},
					nil,
					params.MetricsClient,
					logger,
				),
				serviceConfig.EnableReadVisibilityFromES,
				// frontend visibility never write
				dynamicconfig.GetStringPropertyFn(common.AdvancedVisibilityWritingModeOff),
			), nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &Service{
		Service: baseService,
		config:  serviceConfig,

		numHistoryShards: params.PersistenceConfig.NumHistoryShards,
		bootstrapParams:  params,

		stopC: make(chan struct{}),
	}, nil
}

// Start starts the service
func (s *Service) Start() {

	logger := s.GetLogger()
	logger.Info("frontend starting", tag.Service(common.FrontendServiceName))

	clusterMetadata := s.GetClusterMetadata()

	historyArchiverBootstrapContainer := &archiver.HistoryBootstrapContainer{
		HistoryV2Manager: s.GetHistoryManager(),
		Logger:           s.GetLogger(),
		MetricsClient:    s.GetMetricsClient(),
		ClusterMetadata:  s.GetClusterMetadata(),
		DomainCache:      s.GetDomainCache(),
	}
	visibilityArchiverBootstrapContainer := &archiver.VisibilityBootstrapContainer{
		Logger:          s.GetLogger(),
		MetricsClient:   s.GetMetricsClient(),
		ClusterMetadata: s.GetClusterMetadata(),
		DomainCache:     s.GetDomainCache(),
	}
	err := s.GetArchiverProvider().RegisterBootstrapContainer(
		common.FrontendServiceName,
		historyArchiverBootstrapContainer,
		visibilityArchiverBootstrapContainer,
	)
	if err != nil {
		logger.Fatal("Failed to register archiver bootstrap container", tag.Error(err))
	}

	var replicationMessageSink messaging.Producer
	if clusterMetadata.IsGlobalDomainEnabled() {
		consumerConfig := clusterMetadata.GetReplicationConsumerConfig()
		if consumerConfig != nil && consumerConfig.Type == config.ReplicationConsumerTypeRPC {
			replicationMessageSink = s.GetDomainReplicationQueue()
		} else {
			replicationMessageSink, err = s.GetMessagingClient().NewProducerWithClusterName(
				clusterMetadata.GetCurrentClusterName(),
			)
			if err != nil {
				logger.Fatal("Creating replicationMessageSink producer failed", tag.Error(err))
			}
		}
	} else {
		replicationMessageSink = &mocks.KafkaProducer{}
	}

	wfHandler := NewWorkflowHandler(
		s,
		s.config,
		replicationMessageSink,
	)
	if err != nil {
		logger.Fatal("fail to create workflow handler", tag.Error(err))
	}

	dcRedirectionHandler := NewDCRedirectionHandler(wfHandler, s.bootstrapParams.DCRedirectionPolicy)
	dcRedirectionHandler.RegisterHandler()

	adminHandler := NewAdminHandler(s, s.numHistoryShards, s.bootstrapParams)
	adminHandler.RegisterHandler()

	// must start base service first
	s.Service.Start()
	dcRedirectionHandler.Start()
	adminHandler.Start()

	logger.Info("frontend started", tag.Service(common.FrontendServiceName))

	<-s.stopC
	s.Service.Stop()
}

// Stop stops the service
func (s *Service) Stop() {
	close(s.stopC)
	s.GetLogger().Info("frontend stopped", tag.Service(common.FrontendServiceName))
}
