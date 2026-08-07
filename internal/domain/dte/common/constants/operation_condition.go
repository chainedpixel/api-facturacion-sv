package constants

const (
	Cash = iota + 1
	Credit
	Other
)

var (
	ValidPaymentConditions = []int{
		Cash,
		Credit,
		Other,
	}
)
