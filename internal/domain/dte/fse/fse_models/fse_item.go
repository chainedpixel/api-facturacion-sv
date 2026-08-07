package fse_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/value_objects/financial"
)

type FSEItem struct {
	*models.Item
	Purchase financial.Amount
}

func NewFSEItem(
	baseItem *models.Item,
	purchase financial.Amount,
) *FSEItem {
	return &FSEItem{
		Item:     baseItem,
		Purchase: purchase,
	}
}

func (item *FSEItem) GetPurchase() financial.Amount {
	return item.Purchase
}

func (item *FSEItem) SetPurchase(purchase financial.Amount) {
	item.Purchase = purchase
}
