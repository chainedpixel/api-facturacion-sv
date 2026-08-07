package remission_note_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
)

type RemissionNoteInput struct {
	*models.InputDataCommon
	Receiver         *RemissionNoteReceiver
	Items            []RemissionNoteItem
	RemissionSummary *RemissionNoteSummary
	Extension        interfaces.Extension      `json:"extension,omitempty"`
	Appendixes       []models.Appendix         `json:"appendixes,omitempty"`
	ThirdPartySale   interfaces.ThirdPartySale `json:"thirdPartySale,omitempty"`
}
