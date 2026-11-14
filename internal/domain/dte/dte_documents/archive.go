package dte_documents

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"strings"

	"github.com/chainedpixel/api-facturacion-sv/internal/domain/dte/common/dte_errors"
)

// BuildArchive arma el Archivo DTE de un documento o evento (Normativa v2.0, Anexo I, regla 4.1):
// la estructura de datos firmada, integra y sin alteraciones, mas "firmaElectronica" y "selloRecibido".
//
// La estructura se toma del payload del JWS y no de json_data, porque es exactamente lo que se firmo y
// transmitio (MySQL reordena las llaves de las columnas JSON). Los dos atributos se agregan al final del
// objeto sin reordenar ni reformatear el resto.
func BuildArchive(signedDocument string, receptionStamp *string) (json.RawMessage, error) {
	parts := strings.Split(signedDocument, ".")
	if len(parts) != 3 {
		return nil, dte_errors.NewValidationError("ArchiveNotAvailable", "firmaElectronica")
	}

	payload, err := base64.RawURLEncoding.DecodeString(strings.TrimRight(parts[1], "="))
	if err != nil {
		return nil, dte_errors.NewValidationError("ArchiveNotAvailable", "firmaElectronica")
	}

	structure := bytes.TrimSpace(payload)
	if len(structure) < 2 || structure[0] != '{' || structure[len(structure)-1] != '}' {
		return nil, dte_errors.NewValidationError("ArchiveNotAvailable", "firmaElectronica")
	}

	signature, err := json.Marshal(signedDocument)
	if err != nil {
		return nil, err
	}
	stamp, err := json.Marshal(receptionStamp)
	if err != nil {
		return nil, err
	}

	var archive bytes.Buffer
	archive.Write(bytes.TrimRight(structure[:len(structure)-1], " \t\r\n"))
	if !bytes.HasSuffix(bytes.TrimSpace(structure[:len(structure)-1]), []byte("{")) {
		archive.WriteByte(',')
	}
	archive.WriteString(`"firmaElectronica":`)
	archive.Write(signature)
	archive.WriteString(`,"selloRecibido":`)
	archive.Write(stamp)
	archive.WriteByte('}')

	return archive.Bytes(), nil
}
