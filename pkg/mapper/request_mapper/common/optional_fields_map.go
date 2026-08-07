package common

import (
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/models"
	"github.com/chainedpixel/ordo-factus/pkg/mapper/request_mapper/structs"
)

// CommonOptionalFields groups the optional request fields shared by most DTE document types.
// Use MapCommonOptionalToInputData to map these fields into an InputDataCommon target.
type CommonOptionalFields struct {
	ThirdPartySale *structs.ThirdPartySaleRequest
	Extension      *structs.ExtensionRequest
	OtherDocs      []structs.OtherDocRequest
	RelatedDocs    []structs.RelatedDocRequest
	Appendixes     []structs.AppendixRequest
}

// MapCommonOptionalToInputData maps the common optional request fields into the provided
// InputDataCommon target. It stops and returns the first mapping error encountered.
func MapCommonOptionalToInputData(fields CommonOptionalFields, dest *models.InputDataCommon) error {
	if fields.ThirdPartySale != nil {
		thirdPartySale, err := MapCommonRequestThirdPartySale(fields.ThirdPartySale)
		if err != nil {
			return err
		}
		dest.ThirdPartySale = thirdPartySale
	}

	if fields.Extension != nil {
		extension, err := MapCommonRequestExtension(fields.Extension)
		if err != nil {
			return err
		}
		dest.Extension = extension
	}

	if len(fields.OtherDocs) > 0 {
		otherDocs, err := MapCommonRequestOtherDocuments(fields.OtherDocs)
		if err != nil {
			return err
		}
		dest.OtherDocs = otherDocs
	}

	if len(fields.RelatedDocs) > 0 {
		relatedDocs, err := MapCommonRequestRelatedDocuments(fields.RelatedDocs)
		if err != nil {
			return err
		}
		dest.RelatedDocs = relatedDocs
	}

	if len(fields.Appendixes) > 0 {
		appendixes, err := MapCommonRequestAppendix(fields.Appendixes)
		if err != nil {
			return err
		}
		dest.Appendixes = appendixes
	}

	return nil
}
