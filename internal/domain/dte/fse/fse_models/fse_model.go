package fse_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
)

type FSEModel struct {
	*models.DTEDocument
	FSEItems    []FSEItem
	FSESummary  FSESummary
	FSEReceiver FSEReceiver
}

func NewFSEModel(
	baseDocument *models.DTEDocument,
	items []FSEItem,
	summary FSESummary,
	receiver FSEReceiver,
) *FSEModel {
	return &FSEModel{
		DTEDocument: baseDocument,
		FSEItems:    items,
		FSESummary:  summary,
		FSEReceiver: receiver,
	}
}

func (f *FSEModel) GetItems() []FSEItem {
	return f.FSEItems
}

func (f *FSEModel) GetSummary() FSESummary {
	return f.FSESummary
}

func (f *FSEModel) GetReceiver() FSEReceiver {
	return f.FSEReceiver
}
