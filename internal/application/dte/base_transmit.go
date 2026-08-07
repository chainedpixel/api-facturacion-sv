package dte

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/chainedpixel/ordo-factus/internal/application/ports"
	"github.com/chainedpixel/ordo-factus/internal/domain/dte/transmitter/models"
	"github.com/chainedpixel/ordo-factus/pkg/shared/logs"
)

const (
	MaxRetries     = 2
	MaxTimeout     = 8
	ReceivedStatus = "PROCESADO"
)

// BaseTransmitter encapsulates only the common retransmission logic
type BaseTransmitter struct {
	transmitter ports.DTETransmitter
	signer      ports.SignerManager
}

func NewBaseTransmitter(transmitter ports.DTETransmitter, signer ports.SignerManager) ports.BaseTransmitter {
	return &BaseTransmitter{
		transmitter: transmitter,
		signer:      signer,
	}
}

// RetryTransmission handles the retry and verification logic
func (bt *BaseTransmitter) RetryTransmission(ctx context.Context, document interface{}, token string, nit string) (*models.TransmitResult, error) {
	jsonData, err := json.Marshal(document)
	if err != nil {
		logs.Error("Failed to marshal document for signing", map[string]interface{}{
			"error": err.Error(),
		})
		return nil, err
	}

	signedDoc, err := bt.signer.SignDTE(ctx, jsonData, nit)
	if err != nil {
		logs.Error("Failed to sign document", map[string]interface{}{
			"error": err.Error(),
			"nit":   nit,
		})
		return nil, err
	}

	logs.Info("First attempt to transmit document")
	result, err := bt.transmitter.Transmit(ctx, document, signedDoc, token)
	if err == nil && result.Status == ReceivedStatus {
		logs.Info("Document received on first attempt")
		return result, nil
	}

	if err != nil {
		logs.Error("Failed to transmit document", map[string]interface{}{
			"error": err.Error(),
		})
		return result, err
	}

	logs.Info("Check status of document")
	statusResult, err := bt.CheckStatus(ctx, document, nit)
	if err == nil && statusResult.Status == ReceivedStatus {
		logs.Info("Document already received")
		return statusResult, nil
	}

	logs.Info("Starting retries because document was not received")
	retryCount := 0
	for retryCount < MaxRetries {
		logs.Info(fmt.Sprintf("Retry %d of %d", retryCount+1, MaxRetries), nil)
		result, err = bt.transmitter.Transmit(ctx, document, signedDoc, token)
		if err == nil && result.Status == ReceivedStatus {
			logs.Info("Document received on retry")
			return result, nil
		}

		retryCount++
		time.Sleep(MaxTimeout * time.Second)
	}

	logs.Info("Document was not received")
	return result, err
}

func (bt *BaseTransmitter) CheckStatus(ctx context.Context, document interface{}, nit string) (*models.TransmitResult, error) {
	return bt.transmitter.CheckDocumentStatus(ctx, document, nit)
}
