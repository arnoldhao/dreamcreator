package dependencies

import (
	"context"
	"testing"
	"time"

	"CanMe/backend/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDependencyProvider 模拟依赖提供者
type MockDependencyProvider struct {
	mock.Mock
}

func (m *MockDependencyProvider) GetType() types.DependencyType {
	args := m.Called()
	return args.Get(0).(types.DependencyType)
}

func (m *MockDependencyProvider) Check(ctx context.Context, manager Manager) (*types.DependencyInfo, error) {
	args := m.Called(ctx, manager)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DependencyInfo), args.Error(1)
}

// Fix: Correct Download method signature
func (m *MockDependencyProvider) Download(ctx context.Context, manager Manager, config types.DownloadConfig, progress ProgressCallback) (*types.DependencyInfo, error) {
	args := m.Called(ctx, manager, config, progress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*types.DependencyInfo), args.Error(1)
}

// Fix: Correct Validate method signature
func (m *MockDependencyProvider) Validate(ctx context.Context, execPath string) error {
	args := m.Called(ctx, execPath)
	return args.Error(0)
}

// Fix: Correct ValidateVersion method signature
func (m *MockDependencyProvider) ValidateVersion(ctx context.Context, execPath, expectedVersion string) error {
	args := m.Called(ctx, execPath, expectedVersion)
	return args.Error(0)
}

func (m *MockDependencyProvider) GetDownloadURLWithMirror(version string, mirror string) (string, error) {
	args := m.Called(version, mirror)
	return args.String(0), args.Error(1)
}

func (m *MockDependencyProvider) GetLatestVersionWithMirror(ctx context.Context, manager Manager, mirror string) (string, error) {
	args := m.Called(ctx, manager, mirror)
	return args.String(0), args.Error(1)
}

func TestNewManager(t *testing.T) {
	// Fix: Pass nil for both proxyManager and boltStorage for testing
	manager := NewManager(nil, nil, nil)
	assert.NotNil(t, manager)
	// Remove type assertion, use interface check instead
	assert.Implements(t, (*Manager)(nil), manager)
}

func TestManagerRegister(t *testing.T) {
	// Fix: Pass nil for both proxyManager and boltStorage for testing
	manager := NewManager(nil, nil, nil)
	mockProvider := new(MockDependencyProvider)
	mockProvider.On("GetType").Return(types.DependencyFFmpeg)

	manager.Register(mockProvider)

	// Fix: Update mock expectation to match actual method signature
	mockProvider.On("Check", mock.Anything, mock.Anything).Return(&types.DependencyInfo{
		Type:      types.DependencyFFmpeg,
		Version:   "4.4.0",
		ExecPath:  "/usr/bin/ffmpeg",
		LastCheck: time.Now(),
	}, nil)

	info, err := manager.Get(context.Background(), types.DependencyFFmpeg)
	assert.NoError(t, err)
	assert.NotNil(t, info)
	assert.Equal(t, types.DependencyFFmpeg, info.Type)
	mockProvider.AssertExpectations(t)
}

func TestManagerGetCached(t *testing.T) {
	ctx := context.Background()
	// Fix: Pass nil for both proxyManager and boltStorage for testing
	manager := NewManager(nil, nil, nil)
	mockProvider := new(MockDependencyProvider)

	cachedInfo := &types.DependencyInfo{
		Type:      types.DependencyFFmpeg,
		Version:   "4.4.0",
		ExecPath:  "/usr/bin/ffmpeg",
		LastCheck: time.Now(),
	}

	mockProvider.On("GetType").Return(types.DependencyFFmpeg)
	// Fix: Update mock expectation to match actual method signature with manager parameter
	mockProvider.On("Check", ctx, mock.Anything).Return(cachedInfo, nil).Once()
	manager.Register(mockProvider)

	// First call, will call Check and cache result
	info1, err1 := manager.Get(ctx, types.DependencyFFmpeg)
	assert.NoError(t, err1)
	assert.Equal(t, cachedInfo, info1)

	// Second call should return cached result, not call Check again
	info2, err2 := manager.Get(ctx, types.DependencyFFmpeg)
	assert.NoError(t, err2)
	assert.Equal(t, cachedInfo, info2)

	// Verify Check was only called once
	mockProvider.AssertExpectations(t)
}
