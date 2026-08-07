package fse_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
)

type FSEData struct {
	*models.InputDataCommon
	Items       []FSEItem
	FSESummary  *FSESummary
	FSEReceiver *FSEReceiver
}

func NewFSEData(
	inputDataCommon *models.InputDataCommon,
	items []FSEItem,
	summary *FSESummary,
	receiver *FSEReceiver,
) *FSEData {
	return &FSEData{
		InputDataCommon: inputDataCommon,
		Items:           items,
		FSESummary:      summary,
		FSEReceiver:     receiver,
	}
}

func (data *FSEData) GetItems() []FSEItem {
	return data.Items
}

func (data *FSEData) GetFSESummary() *FSESummary {
	return data.FSESummary
}

func (data *FSEData) GetFSEReceiver() *FSEReceiver {
	return data.FSEReceiver
}

func (data *FSEData) SetItems(items []FSEItem) {
	data.Items = items
}

func (data *FSEData) SetFSESummary(summary *FSESummary) {
	data.FSESummary = summary
}

func (data *FSEData) SetFSEReceiver(receiver *FSEReceiver) {
	data.FSEReceiver = receiver
}
