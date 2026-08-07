package dte_documents

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/domain/auth"
	"github.com/chainedpixel/ordo-factus/internal/domain/ports"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"

	"github.com/chainedpixel/ordo-factus/pkg/shared/shared_error"
	"github.com/chainedpixel/ordo-factus/pkg/shared/utils"
)

type sequentialNumberService struct {
	sequentialRepo  ports.SequentialNumberRepositoryPort
	authRepo        auth.AuthRepositoryPort
	reservationRepo ReservedSequenceRepositoryPort
}

func NewSequentialNumberService(
	sequentialRepo ports.SequentialNumberRepositoryPort,
	authRepo auth.AuthRepositoryPort,
	reservationRepo ReservedSequenceRepositoryPort,
) SequentialNumberManager {
	return &sequentialNumberService{
		sequentialRepo:  sequentialRepo,
		authRepo:        authRepo,
		reservationRepo: reservationRepo,
	}
}

func (m *sequentialNumberService) GetNextControlNumber(ctx context.Context, dteType string, branchID uint, posCode, establishmentCode *string) (string, error) {
	defaultValue := "0000"
	if posCode == nil {
		posCode = &defaultValue
	}
	if establishmentCode == nil {
		establishmentCode = &defaultValue
	}

	controlNumber, _, err := m.ReserveNextNumber(ctx, dteType, branchID, posCode, establishmentCode, "", false)
	if err != nil {
		return "", err
	}

	return controlNumber, nil
}

func (m *sequentialNumberService) ReserveNextNumber(ctx context.Context, dteType string, branchID uint, posCode, establishmentCode *string, _ string, isContingency bool) (string, uint, error) {
	defaultValue := "0000"
	if posCode == nil {
		posCode = &defaultValue
	}
	if establishmentCode == nil {
		establishmentCode = &defaultValue
	}

	user, err := m.authRepo.GetByBranchID(ctx, branchID)
	if err != nil {
		return "", 0, shared_error.NewGeneralServiceError("SequentialNumberManager", "ReserveNextNumber", "failed to get user by branchID", err)
	}

	year := utils.TimeNow().Year()

	releasedNumber, err := m.reservationRepo.GetOldestReleasedNumber(ctx, branchID, dteType, year)
	if err == nil && releasedNumber != nil {
		logs.Info("Reusing released sequence number", map[string]interface{}{
			"sequenceNumber": releasedNumber.SequenceNumber,
			"branchID":       branchID,
			"dteType":        dteType,
			"year":           year,
		})

		var expiresAt *time.Time
		if !isContingency {
			expiration := utils.TimeNow().Add(1 * time.Hour)
			expiresAt = &expiration
		}

		markErr := m.reservationRepo.MarkAsReserved(ctx, releasedNumber.ID, expiresAt)
		if markErr == nil {
			controlNumber := m.formatControlNumber(user.YearInDTE, dteType, *establishmentCode, *posCode, releasedNumber.SequenceNumber, year)
			return controlNumber, releasedNumber.ID, nil
		}

		logs.Info("Released number was claimed by concurrent request, generating new number", map[string]interface{}{
			"reservationID": releasedNumber.ID,
			"branchID":      branchID,
		})
	}

	logs.Info("No released numbers found, generating new sequence number", map[string]interface{}{
		"branchID": branchID,
		"dteType":  dteType,
		"year":     year,
	})

	correlativeNumber, err := m.sequentialRepo.GetNext(ctx, dteType, branchID)
	if err != nil {
		return "", 0, shared_error.NewGeneralServiceError("SequentialNumberManager", "ReserveNextNumber", "failed to get next control number", err)
	}

	var expiresAt *time.Time
	if !isContingency {
		expiration := utils.TimeNow().Add(1 * time.Hour)
		expiresAt = &expiration
	}

	reservation := &ReservedSequence{
		BranchID:       branchID,
		DTEType:        dteType,
		SequenceNumber: uint(correlativeNumber),
		Year:           year,
		Status:         ReservationStatusReserved,
		ReservedAt:     utils.TimeNow(),
		ExpiresAt:      expiresAt,
		IsContingency:  isContingency,
	}

	reservationID, err := m.reservationRepo.Create(ctx, reservation)
	if err != nil {
		return "", 0, shared_error.NewGeneralServiceError("SequentialNumberManager", "ReserveNextNumber", "failed to create reservation", err)
	}

	controlNumber := m.formatControlNumber(user.YearInDTE, dteType, *establishmentCode, *posCode, uint(correlativeNumber), year)
	return controlNumber, reservationID, nil
}

func (m *sequentialNumberService) ConfirmReservation(ctx context.Context, controlNumber string, documentID string, branchID uint) error {
	dteType, seqNum, year, err := m.parseControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ConfirmReservation", "failed to parse control number", err)
	}

	err = m.reservationRepo.UpdateStatusWithDocumentID(ctx, branchID, dteType, seqNum, year, ReservationStatusConfirmed, utils.TimeNow(), documentID)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ConfirmReservation", "failed to update status", err)
	}

	return nil
}

func (m *sequentialNumberService) ReleaseReservation(ctx context.Context, controlNumber string, rejectionReason string, haciendaCode string, branchID uint) error {
	dteType, seqNum, year, err := m.parseControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ReleaseReservation", "failed to parse control number", err)
	}

	err = m.reservationRepo.UpdateStatusWithReason(ctx, branchID, dteType, seqNum, year, ReservationStatusReleased, utils.TimeNow(), rejectionReason, haciendaCode)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ReleaseReservation", "failed to update status", err)
	}

	return nil
}

func (m *sequentialNumberService) ConfirmReservationByDocumentID(ctx context.Context, documentID string) error {
	reservation, err := m.reservationRepo.GetByDocumentID(ctx, documentID)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ConfirmReservationByDocumentID", "failed to get reservation", err)
	}

	err = m.reservationRepo.UpdateStatus(ctx, reservation.BranchID, reservation.DTEType, reservation.SequenceNumber, reservation.Year, ReservationStatusConfirmed, utils.TimeNow())
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ConfirmReservationByDocumentID", "failed to update status", err)
	}

	return nil
}

func (m *sequentialNumberService) ReleaseReservationByDocumentID(ctx context.Context, documentID string, rejectionReason string, haciendaCode string) error {
	reservation, err := m.reservationRepo.GetByDocumentID(ctx, documentID)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ReleaseReservationByDocumentID", "failed to get reservation", err)
	}

	err = m.reservationRepo.UpdateStatusWithReason(ctx, reservation.BranchID, reservation.DTEType, reservation.SequenceNumber, reservation.Year, ReservationStatusReleased, utils.TimeNow(), rejectionReason, haciendaCode)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "ReleaseReservationByDocumentID", "failed to update status", err)
	}

	return nil
}

func (m *sequentialNumberService) MarkReservationAsContingency(ctx context.Context, reservationID uint, documentID string, contingencyDocID string) error {
	err := m.reservationRepo.UpdateAsContingency(ctx, reservationID, documentID, contingencyDocID)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "MarkReservationAsContingency", "failed to update as contingency", err)
	}

	return nil
}

func (m *sequentialNumberService) MarkReservationAsContingencyByControlNumber(ctx context.Context, controlNumber string, documentID string, contingencyDocID string, branchID uint) error {
	dteType, seqNum, year, err := m.parseControlNumber(controlNumber)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "MarkReservationAsContingencyByControlNumber", "failed to parse control number", err)
	}

	err = m.reservationRepo.UpdateAsContingencyByControlNumber(ctx, branchID, dteType, seqNum, year, documentID, contingencyDocID)
	if err != nil {
		return shared_error.NewGeneralServiceError("SequentialNumberManager", "MarkReservationAsContingencyByControlNumber", "failed to update as contingency", err)
	}

	return nil
}

func (m *sequentialNumberService) formatControlNumber(yearInDTE bool, dteType, establishmentCode, posCode string, seqNum uint, year int) string {
	if yearInDTE {
		return fmt.Sprintf("DTE-%s-%s%s-%s%011d", dteType, establishmentCode, posCode, strconv.Itoa(year), seqNum)
	}
	return fmt.Sprintf("DTE-%s-%s%s-%015d", dteType, establishmentCode, posCode, seqNum)
}

func (m *sequentialNumberService) parseControlNumber(controlNumber string) (dteType string, seqNum uint, year int, err error) {
	pattern := `^DTE-(\d{2})-([A-Z0-9]+)-(\d{15})$`
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(controlNumber)

	if len(matches) != 4 {
		err = fmt.Errorf("invalid control number format: expected DTE-XX-XXXXXXXX-XXXXXXXXXXXXXXX, got: %s", controlNumber)
		return
	}

	dteType = matches[1]
	numericPart := matches[3]

	possibleYear, parseErr := strconv.Atoi(numericPart[:4])
	if parseErr == nil && possibleYear >= 2020 && possibleYear <= 2099 {
		year = possibleYear
		seqNumInt, seqErr := strconv.Atoi(numericPart[4:])
		if seqErr != nil {
			err = fmt.Errorf("failed to parse sequence number from year-prefixed format: %w", seqErr)
			return
		}
		seqNum = uint(seqNumInt)
	} else {
		seqNumInt, seqErr := strconv.Atoi(numericPart)
		if seqErr != nil {
			err = fmt.Errorf("failed to parse sequence number: %w", seqErr)
			return
		}
		seqNum = uint(seqNumInt)
		year = utils.TimeNow().Year()
	}

	return
}
