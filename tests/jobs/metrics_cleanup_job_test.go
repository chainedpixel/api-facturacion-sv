package jobs

import (
	"errors"
	"testing"
	"time"

	test "github.com/chainedpixel/ordo-factus/tests"
	"github.com/chainedpixel/ordo-factus/tests/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"github.com/chainedpixel/ordo-factus/internal/infrastructure/jobs"
)

func TestMetricsCleanupJob_Execute(t *testing.T) {
	test.TestMain(t)

	type mockSetup struct {
		cache func(*mocks.MockCacheManager)
	}

	cases := []struct {
		name  string
		setup mockSetup
	}{
		{
			name: "success – cleans both duration and counter keys",
			setup: mockSetup{
				cache: func(m *mocks.MockCacheManager) {
					m.EXPECT().ScanKeys("metrics:*:durations").Return([]string{
						"metrics:0614:POST:invoices:durations",
						"metrics:0614:GET:dte:durations",
					}, nil)
					m.EXPECT().Expire("metrics:0614:POST:invoices:durations", 24*time.Hour).Return(nil)
					m.EXPECT().Expire("metrics:0614:GET:dte:durations", 24*time.Hour).Return(nil)

					m.EXPECT().ScanKeys("metrics:*:counters").Return([]string{
						"metrics:0614:POST:invoices:counters",
					}, nil)
					m.EXPECT().Expire("metrics:0614:POST:invoices:counters", 24*time.Hour).Return(nil)
				},
			},
		},
		{
			name: "success – no keys found",
			setup: mockSetup{
				cache: func(m *mocks.MockCacheManager) {
					m.EXPECT().ScanKeys("metrics:*:durations").Return([]string{}, nil)
					m.EXPECT().ScanKeys("metrics:*:counters").Return([]string{}, nil)
				},
			},
		},
		{
			name: "error on scan durations – continues with counters",
			setup: mockSetup{
				cache: func(m *mocks.MockCacheManager) {
					m.EXPECT().ScanKeys("metrics:*:durations").Return(nil, errors.New("redis connection error"))
					m.EXPECT().ScanKeys("metrics:*:counters").Return([]string{
						"metrics:0614:POST:invoices:counters",
					}, nil)
					m.EXPECT().Expire("metrics:0614:POST:invoices:counters", 24*time.Hour).Return(nil)
				},
			},
		},
		{
			name: "error on scan counters – durations still processed",
			setup: mockSetup{
				cache: func(m *mocks.MockCacheManager) {
					m.EXPECT().ScanKeys("metrics:*:durations").Return([]string{
						"metrics:0614:POST:invoices:durations",
					}, nil)
					m.EXPECT().Expire("metrics:0614:POST:invoices:durations", 24*time.Hour).Return(nil)
					m.EXPECT().ScanKeys("metrics:*:counters").Return(nil, errors.New("redis connection error"))
				},
			},
		},
		{
			name: "partial expire failure – continues with remaining keys",
			setup: mockSetup{
				cache: func(m *mocks.MockCacheManager) {
					m.EXPECT().ScanKeys("metrics:*:durations").Return([]string{
						"metrics:0614:POST:invoices:durations",
						"metrics:0614:GET:dte:durations",
					}, nil)
					m.EXPECT().Expire("metrics:0614:POST:invoices:durations", 24*time.Hour).Return(errors.New("expire failed"))
					m.EXPECT().Expire("metrics:0614:GET:dte:durations", 24*time.Hour).Return(nil)

					m.EXPECT().ScanKeys("metrics:*:counters").Return([]string{}, nil)
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockCache := mocks.NewMockCacheManager(ctrl)
			tc.setup.cache(mockCache)

			job := jobs.NewMetricsCleanupJob(mockCache)
			job.Execute()
		})
	}
}

func TestMetricsCleanupJob_ConcurrentExecution(t *testing.T) {
	test.TestMain(t)

	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockCache := mocks.NewMockCacheManager(ctrl)

	blockCh := make(chan struct{})
	mockCache.EXPECT().ScanKeys("metrics:*:durations").DoAndReturn(func(pattern string) ([]string, error) {
		<-blockCh
		return []string{}, nil
	})
	mockCache.EXPECT().ScanKeys("metrics:*:counters").Return([]string{}, nil).MaxTimes(1)

	job := jobs.NewMetricsCleanupJob(mockCache)

	go job.Execute()

	time.Sleep(50 * time.Millisecond)
	assert.True(t, job.IsRunning.Load(), "job should be marked as running")

	job.Execute()

	close(blockCh)
	time.Sleep(50 * time.Millisecond)
	assert.False(t, job.IsRunning.Load(), "job should be marked as not running after completion")
}
