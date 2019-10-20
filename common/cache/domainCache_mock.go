// The MIT License (MIT)
//
// Copyright (c) 2019 Uber Technologies, Inc.
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.
//

// Code generated by MockGen. DO NOT EDIT.
// Source: domainCache.go

// Package cache is a generated GoMock package.
package cache

import (
	gomock "github.com/golang/mock/gomock"
	cache "github.com/uber/cadence/common/cache"
	reflect "reflect"
)

// MockDomainCache is a mock of DomainCache interface
type MockDomainCache struct {
	ctrl     *gomock.Controller
	recorder *MockDomainCacheMockRecorder
}

// MockDomainCacheMockRecorder is the mock recorder for MockDomainCache
type MockDomainCacheMockRecorder struct {
	mock *MockDomainCache
}

// NewMockDomainCache creates a new mock instance
func NewMockDomainCache(ctrl *gomock.Controller) *MockDomainCache {
	mock := &MockDomainCache{ctrl: ctrl}
	mock.recorder = &MockDomainCacheMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use
func (m *MockDomainCache) EXPECT() *MockDomainCacheMockRecorder {
	return m.recorder
}

// Start mocks base method
func (m *MockDomainCache) Start() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Start")
}

// Start indicates an expected call of Start
func (mr *MockDomainCacheMockRecorder) Start() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Start", reflect.TypeOf((*MockDomainCache)(nil).Start))
}

// Stop mocks base method
func (m *MockDomainCache) Stop() {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Stop")
}

// Stop indicates an expected call of Stop
func (mr *MockDomainCacheMockRecorder) Stop() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Stop", reflect.TypeOf((*MockDomainCache)(nil).Stop))
}

// RegisterDomainChangeCallback mocks base method
func (m *MockDomainCache) RegisterDomainChangeCallback(shard int, initialNotificationVersion int64, prepareCallback cache.PrepareCallbackFn, callback cache.CallbackFn) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "RegisterDomainChangeCallback", shard, initialNotificationVersion, prepareCallback, callback)
}

// RegisterDomainChangeCallback indicates an expected call of RegisterDomainChangeCallback
func (mr *MockDomainCacheMockRecorder) RegisterDomainChangeCallback(shard, initialNotificationVersion, prepareCallback, callback interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RegisterDomainChangeCallback", reflect.TypeOf((*MockDomainCache)(nil).RegisterDomainChangeCallback), shard, initialNotificationVersion, prepareCallback, callback)
}

// UnregisterDomainChangeCallback mocks base method
func (m *MockDomainCache) UnregisterDomainChangeCallback(shard int) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "UnregisterDomainChangeCallback", shard)
}

// UnregisterDomainChangeCallback indicates an expected call of UnregisterDomainChangeCallback
func (mr *MockDomainCacheMockRecorder) UnregisterDomainChangeCallback(shard interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UnregisterDomainChangeCallback", reflect.TypeOf((*MockDomainCache)(nil).UnregisterDomainChangeCallback), shard)
}

// GetDomain mocks base method
func (m *MockDomainCache) GetDomain(name string) (*cache.DomainCacheEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomain", name)
	ret0, _ := ret[0].(*cache.DomainCacheEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomain indicates an expected call of GetDomain
func (mr *MockDomainCacheMockRecorder) GetDomain(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomain", reflect.TypeOf((*MockDomainCache)(nil).GetDomain), name)
}

// GetDomainByID mocks base method
func (m *MockDomainCache) GetDomainByID(id string) (*cache.DomainCacheEntry, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomainByID", id)
	ret0, _ := ret[0].(*cache.DomainCacheEntry)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomainByID indicates an expected call of GetDomainByID
func (mr *MockDomainCacheMockRecorder) GetDomainByID(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomainByID", reflect.TypeOf((*MockDomainCache)(nil).GetDomainByID), id)
}

// GetDomainID mocks base method
func (m *MockDomainCache) GetDomainID(name string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomainID", name)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomainID indicates an expected call of GetDomainID
func (mr *MockDomainCacheMockRecorder) GetDomainID(name interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomainID", reflect.TypeOf((*MockDomainCache)(nil).GetDomainID), name)
}

// GetDomainName mocks base method
func (m *MockDomainCache) GetDomainName(id string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetDomainName", id)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetDomainName indicates an expected call of GetDomainName
func (mr *MockDomainCacheMockRecorder) GetDomainName(id interface{}) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetDomainName", reflect.TypeOf((*MockDomainCache)(nil).GetDomainName), id)
}

// GetAllDomain mocks base method
func (m *MockDomainCache) GetAllDomain() map[string]*cache.DomainCacheEntry {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAllDomain")
	ret0, _ := ret[0].(map[string]*cache.DomainCacheEntry)
	return ret0
}

// GetAllDomain indicates an expected call of GetAllDomain
func (mr *MockDomainCacheMockRecorder) GetAllDomain() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAllDomain", reflect.TypeOf((*MockDomainCache)(nil).GetAllDomain))
}

// GetCacheSize mocks base method
func (m *MockDomainCache) GetCacheSize() (int64, int64) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetCacheSize")
	ret0, _ := ret[0].(int64)
	ret1, _ := ret[1].(int64)
	return ret0, ret1
}

// GetCacheSize indicates an expected call of GetCacheSize
func (mr *MockDomainCacheMockRecorder) GetCacheSize() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetCacheSize", reflect.TypeOf((*MockDomainCache)(nil).GetCacheSize))
}
