package remission_note_models

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/interfaces"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
)

type RemissionNoteReceiver struct {
	*models.Receiver
	BienTitulo *string `json:"bienTitulo"`
}

func (r *RemissionNoteReceiver) GetBienTitulo() *string {
	return r.BienTitulo
}

func (r *RemissionNoteReceiver) SetBienTitulo(bienTitulo *string) error {
	r.BienTitulo = bienTitulo
	return nil
}

// Ensure it implements the base interface
var _ interfaces.Receiver = (*RemissionNoteReceiver)(nil)
