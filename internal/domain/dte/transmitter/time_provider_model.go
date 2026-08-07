package transmitter

import (
	"time"

	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

// RealTimeProvider es un proveedor de tiempo real
type RealTimeProvider struct{}

func (p *RealTimeProvider) Now() time.Time {
	return utils.TimeNow()
}

func (p *RealTimeProvider) Sleep(d time.Duration) {
	time.Sleep(d)
}
