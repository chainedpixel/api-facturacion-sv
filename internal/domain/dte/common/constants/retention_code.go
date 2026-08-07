package constants

import "github.com/shopspring/decimal"

const (
	RetentionOnePercent      = "22"
	RetentionThirteenPercent = "C4"
	OtherRetentions          = "C9"
)

var (
	AllowedRetentionCodes = map[string]bool{
		RetentionOnePercent:      true,
		RetentionThirteenPercent: true,
		OtherRetentions:          true,
	}

	ListRetentionCodes = []string{
		RetentionOnePercent,
		RetentionThirteenPercent,
	}

	GetRetentionAmount = map[string]decimal.Decimal{
		RetentionOnePercent:      decimal.NewFromFloat(0.01),
		RetentionThirteenPercent: decimal.NewFromFloat(0.13),
	}
)
