package dte

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth/models"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/dte"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/common/constants"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/dte_documents"
	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
)

type DTEConsultUseCase struct {
	dteService dte_documents.DTEManager
}

func NewDTEConsultUseCase(dteService dte_documents.DTEManager) *DTEConsultUseCase {
	return &DTEConsultUseCase{
		dteService: dteService,
	}
}

func (u *DTEConsultUseCase) GetByGenerationCode(ctx context.Context, id string) (interface{}, error) {
	claims := ctx.Value("claims").(*models.AuthClaims)

	dte, err := u.dteService.GetByGenerationCodeConsult(ctx, claims.BranchID, id)
	if err != nil {
		return nil, err
	}

	return dte, nil
}

func (u *DTEConsultUseCase) GetAllDTEs(ctx context.Context, r *http.Request) (*dte.DTEListResponse, error) {
	filters, err := parseDTEFilters(r)
	if err != nil {
		return nil, err
	}

	response, err := u.dteService.GetAllDTEs(ctx, filters)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func parseDTEFilters(r *http.Request) (*dte.DTEFilters, error) {
	filters := &dte.DTEFilters{
		IncludeAll: r.URL.Query().Get("all") == "true",
	}

	if !filters.IncludeAll {
		filters.BranchID = r.Context().Value("claims").(*models.AuthClaims).BranchID
	}

	startDateStr := r.URL.Query().Get("startDate")
	endDateStr := r.URL.Query().Get("endDate")

	var startDate, endDate *time.Time
	if startDateStr != "" {
		parsedStartDate, err := time.Parse(time.RFC3339, startDateStr)
		if err != nil {
			return nil, shared_error.NewFormattedGeneralServiceError("ListDTEsUseCase", "parseDTEFilters", "InvalidQueryParam", "startDate", "0000-00-00")
		}
		startDate = &parsedStartDate
	}

	if endDateStr != "" {
		parsedEndDate, err := time.Parse(time.RFC3339, endDateStr)
		if err != nil {
			return nil, shared_error.NewFormattedGeneralServiceError("ListDTEsUseCase", "parseDTEFilters", "InvalidQueryParam", "endDate", "0000-00-00")
		}
		endDate = &parsedEndDate
	}

	if status := r.URL.Query().Get("status"); status != "" {
		if !constants.ValidReceiverDocumentStates[strings.ToUpper(status)] {
			return nil, shared_error.NewFormattedGeneralServiceError("ListDTEsUseCase", "parseDTEFilters", "InvalidQueryParam", "status", "'received', 'invalidated', 'rejected'", nil)
		} else {
			filters.Status = strings.ToUpper(status)
		}
	}

	if pageStr := r.URL.Query().Get("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			filters.Page = page
		} else {
			filters.Page = 1
		}
	} else {
		filters.Page = 1
	}

	if transmission := r.URL.Query().Get("transmission"); transmission != "" {
		if !constants.ValidTransmissionTypes[strings.ToUpper(transmission)] {
			return nil, shared_error.NewFormattedGeneralServiceError("ListDTEsUseCase", "parseDTEFilters", "InvalidQueryParam", "transmission", "'normal', 'contingency'", nil)
		} else {
			filters.Transmission = strings.ToUpper(transmission)
		}
	}

	if pageSizeStr := r.URL.Query().Get("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			filters.PageSize = pageSize
		} else {
			filters.PageSize = 5
		}
	} else {
		filters.PageSize = 5
	}

	if dteType := r.URL.Query().Get("type"); dteType != "" {
		if strings.Contains(dteType, ",") {
			types := strings.Split(dteType, ",")
			for i, t := range types {
				types[i] = strings.TrimSpace(t)
				if !constants.ValidDTETypes[types[i]] {
					return nil, shared_error.NewFormattedGeneralServiceError("ListDTEsUseCase", "parseDTEFilters", "InvalidQueryParam", "type", "01-15")
				}
			}
			filters.DTETypes = types
		} else {
			if !constants.ValidDTETypes[dteType] {
				return nil, shared_error.NewFormattedGeneralServiceError("ListDTEsUseCase", "parseDTEFilters", "InvalidQueryParam", "type", "01-15")
			} else {
				filters.DTEType = dteType
				filters.DTETypes = []string{dteType}
			}
		}
	}

	filters.StartDate = startDate
	filters.EndDate = endDate

	return filters, nil
}
