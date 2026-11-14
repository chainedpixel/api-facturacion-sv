package strategy

import (
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/calendar"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/dte_errors"
)

const (
	remissionNoteGraceDays = 3
	adjustmentWindowMonths = 3
)

var allowedRelatedDocTypes = map[string]map[string]bool{
	constants.NotaRemisionElectronica: constants.ValidRemissionNoteDTETypesRelateDoc,
}

// ValidateRelatedDocType validates the related document type allowed for the DTE type being issued.
func ValidateRelatedDocType(dteType, relatedType string) *dte_errors.DTEError {
	allowed, ok := allowedRelatedDocTypes[dteType]
	if !ok || allowed[relatedType] {
		return nil
	}
	return dte_errors.NewDTEErrorSimple("InvalidRelatedDocDTEType", relatedType, constants.ShowValidRelatedDocTypes(allowed))
}

// ValidateRelatedDocPeriod validates the generation window against the related document: a remission note
// allows its tax period plus three days, a DCL allows its tax period, and a CCF or CR adjusted by a credit
// or debit note allows three months from its generation. An unknown related date is not validated.
func ValidateRelatedDocPeriod(dteType, relatedType string, relatedDate, generation time.Time) *dte_errors.DTEError {
	if relatedDate.IsZero() {
		return nil
	}

	var deadline time.Time

	switch {
	case relatedType == constants.NotaRemisionElectronica:
		deadline = endOfMonth(relatedDate).AddDate(0, 0, remissionNoteGraceDays)
	case relatedType == constants.DocContableLiquidacionElectronico:
		deadline = endOfMonth(relatedDate)
	case constants.ValidAdjustmentDTETypes[relatedType] &&
		(dteType == constants.NotaCreditoElectronica || dteType == constants.NotaDebitoElectronica):
		deadline = calendar.AddMonthsClamped(relatedDate, adjustmentWindowMonths)
	default:
		return nil
	}

	if generation.Format("2006-01-02") > deadline.In(generation.Location()).Format("2006-01-02") {
		return dte_errors.NewDTEErrorSimple("RelatedDocOutOfPeriod", relatedType,
			relatedDate.Format("2006-01-02"), deadline.Format("2006-01-02"))
	}
	return nil
}

func endOfMonth(day time.Time) time.Time {
	return time.Date(day.Year(), day.Month(), 1, 0, 0, 0, 0, day.Location()).AddDate(0, 1, -1)
}
