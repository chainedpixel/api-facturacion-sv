package email

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"strconv"
	texttemplate "text/template"
	"time"

	mails "github.com/chainedpixel/ordo-factus/assets/mails"
	"github.com/chainedpixel/ordo-factus/config"
	"github.com/chainedpixel/ordo-factus/internal/domain/core/event"
)

var contingencyTypeLabels = map[int]string{
	1: "Servicio del Ministerio de Hacienda no disponible",
	2: "Error de conexión con sistemas internos",
	3: "Falla en el servicio de internet",
	4: "Interrupción del servicio eléctrico",
	5: "Error en el proceso de emisión del documento",
}

var dteTypeLabels = map[string]string{
	"01": "Factura Electrónica",
	"03": "Comprobante de Crédito Fiscal Electrónico",
	"04": "Nota de Remisión Electrónica",
	"05": "Nota de Crédito Electrónica",
	"06": "Nota de Débito Electrónica",
	"07": "Comprobante de Retención Electrónico",
	"08": "Comprobante de Liquidación Electrónico",
	"09": "Documento Contable de Liquidación Electrónico",
	"11": "Factura de Exportación Electrónica",
	"14": "Factura de Sujeto Excluido Electrónica",
	"15": "Comprobante de Donación Electrónico",
}

type TemplateRenderer struct {
	icons        map[string]htmltemplate.HTML
	htmlByEvent  map[string]*htmltemplate.Template
	plainByEvent map[string]*texttemplate.Template
	subjectByEvt map[string]string
	badgeByEvt   map[string]string
	titleByEvt   map[string]string
	introByEvt   map[string]string
	iconByEvt    map[string]string
}

func NewTemplateRenderer() (*TemplateRenderer, error) {
	r := &TemplateRenderer{
		icons:        make(map[string]htmltemplate.HTML),
		htmlByEvent:  make(map[string]*htmltemplate.Template),
		plainByEvent: make(map[string]*texttemplate.Template),
	}

	for _, name := range []string{"icon_warning", "icon_error", "icon_info"} {
		b, err := mails.FS.ReadFile("partials/" + name + ".svg")
		if err != nil {
			return nil, fmt.Errorf("read svg %s: %w", name, err)
		}
		r.icons[name] = htmltemplate.HTML(string(b))
	}

	layoutBytes, err := mails.FS.ReadFile("layout.html")
	if err != nil {
		return nil, fmt.Errorf("read layout.html: %w", err)
	}

	contentTemplates := map[string]string{
		event.EventContingencyActivated:    "contingency_activated.html",
		event.EventEmissionFailure:         "emission_failure.html",
		event.EventRetransmissionJobFailed: "retransmission_job_failed.html",
	}
	for evtName, path := range contentTemplates {
		b, err := mails.FS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		t, err := htmltemplate.New("layout.html").Parse(string(layoutBytes))
		if err != nil {
			return nil, fmt.Errorf("parse layout.html: %w", err)
		}
		if _, err := t.Parse(string(b)); err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		r.htmlByEvent[evtName] = t
	}

	plainFiles := map[string]string{
		event.EventContingencyActivated:    "plain/contingency_activated.txt",
		event.EventEmissionFailure:         "plain/emission_failure.txt",
		event.EventRetransmissionJobFailed: "plain/retransmission_job_failed.txt",
	}
	for evtName, path := range plainFiles {
		b, err := mails.FS.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		tt, err := texttemplate.New(evtName).Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", path, err)
		}
		r.plainByEvent[evtName] = tt
	}

	r.subjectByEvt = map[string]string{
		event.EventContingencyActivated:    "[Ordo-Factus] Contingencia activada en sucursal #%d",
		event.EventEmissionFailure:         "[Ordo-Factus] DTE %s falla tras %d intento(s)",
		event.EventRetransmissionJobFailed: "[Ordo-Factus] Job de retransmisión: fallo de ejecución",
	}
	r.badgeByEvt = map[string]string{
		event.EventContingencyActivated:    "Contingencia",
		event.EventEmissionFailure:         "Emisión",
		event.EventRetransmissionJobFailed: "Job",
	}
	r.titleByEvt = map[string]string{
		event.EventContingencyActivated:    "Contingencia activada",
		event.EventEmissionFailure:         "Falla consistente de emisión",
		event.EventRetransmissionJobFailed: "Job de retransmisión con fallos",
	}
	r.introByEvt = map[string]string{
		event.EventContingencyActivated:    "El sistema detectó una contingencia y activó el modo offline para la sucursal afectada.",
		event.EventEmissionFailure:         "Un DTE no pudo emitirse después de varios intentos. Requiere atención.",
		event.EventRetransmissionJobFailed: "El job programado de retransmisión finalizó con documentos fallidos.",
	}
	r.iconByEvt = map[string]string{
		event.EventContingencyActivated:    "icon_warning",
		event.EventEmissionFailure:         "icon_error",
		event.EventRetransmissionJobFailed: "icon_warning",
	}

	return r, nil
}

func (r *TemplateRenderer) Render(evt event.Event) (string, string, string, error) {
	htmlTpl, ok := r.htmlByEvent[evt.Name()]
	if !ok {
		return "", "", "", fmt.Errorf("no html template for event %s", evt.Name())
	}
	plainTpl, ok := r.plainByEvent[evt.Name()]
	if !ok {
		return "", "", "", fmt.Errorf("no plain template for event %s", evt.Name())
	}

	subject, err := r.buildSubject(evt)
	if err != nil {
		return "", "", "", err
	}

	data := r.buildData(evt, subject)

	var plainBuf bytes.Buffer
	if err := plainTpl.Execute(&plainBuf, data); err != nil {
		return "", "", "", fmt.Errorf("render plain: %w", err)
	}

	var htmlBuf bytes.Buffer
	if err := htmlTpl.ExecuteTemplate(&htmlBuf, "layout.html", data); err != nil {
		return "", "", "", fmt.Errorf("render html: %w", err)
	}

	return subject, htmlBuf.String(), plainBuf.String(), nil
}

func (r *TemplateRenderer) buildSubject(evt event.Event) (string, error) {
	pattern, ok := r.subjectByEvt[evt.Name()]
	if !ok {
		return "", fmt.Errorf("no subject pattern for event %s", evt.Name())
	}
	switch e := evt.(type) {
	case event.ContingencyActivatedEvent:
		return fmt.Sprintf(pattern, e.BranchID), nil
	case event.EmissionFailureEvent:
		id := e.ControlNumber
		if id == "" {
			id = e.GenerationCode
		}
		return fmt.Sprintf(pattern, id, e.Attempts), nil
	case event.RetransmissionJobFailedEvent:
		if len(e.FailedDocuments) > 0 {
			return fmt.Sprintf("[Ordo-Factus] Retransmisión: %d documento(s) rechazado(s) por MH", len(e.FailedDocuments)), nil
		}
		return pattern, nil
	}
	return pattern, nil
}

func (r *TemplateRenderer) buildData(evt event.Event, subject string) map[string]any {
	data := make(map[string]any, 32)
	for k, v := range evt.Payload() {
		data[k] = v
	}
	aliases := map[string]string{
		"branch_id":             "BranchID",
		"branch_address":        "BranchAddress",
		"client_name":           "ClientName",
		"contingency_type":      "ContingencyType",
		"reason":                "Reason",
		"affected_docs":         "AffectedDocs",
		"nit":                   "NIT",
		"dte_type":              "DTEType",
		"control_number":        "ControlNumber",
		"error_code":            "ErrorCode",
		"last_error":            "LastError",
		"hacienda_observations": "HaciendaObservations",
		"attempts":              "Attempts",
		"job_name":              "JobName",
		"failed_documents":      "FailedDocuments",
	}
	for snake, pascal := range aliases {
		if v, ok := data[snake]; ok {
			data[pascal] = v
		}
	}

	if raw, ok := data["contingency_type"].(string); ok {
		if n, err := strconv.Atoi(raw); err == nil {
			if label, exists := contingencyTypeLabels[n]; exists {
				data["ContingencyTypeLabel"] = label
			}
		}
	}
	if raw, ok := data["dte_type"].(string); ok {
		if label, exists := dteTypeLabels[raw]; exists {
			data["DTETypeLabel"] = label
		}
	}

	data["Subject"] = subject
	data["Badge"] = r.badgeByEvt[evt.Name()]
	data["Title"] = r.titleByEvt[evt.Name()]
	data["Intro"] = r.introByEvt[evt.Name()]
	data["Icon"] = r.icons[r.iconByEvt[evt.Name()]]
	data["APIVersion"] = currentAPIVersion()
	data["Environment"] = currentEnvironment()
	data["Timestamp"] = formatTimestamp(evt.OccurredAt())
	data["AggregateID"] = evt.AggregateID()
	return data
}

func currentAPIVersion() string {
	if config.Server != nil && config.Server.APIVersion != "" {
		return config.Server.APIVersion
	}
	return "3.0.0"
}

func currentEnvironment() string {
	if config.Server == nil || config.Server.AmbientCode == "" {
		return "unknown"
	}
	switch config.Server.AmbientCode {
	case "00":
		return "test"
	case "01":
		return "production"
	}
	return config.Server.AmbientCode
}

func formatTimestamp(t time.Time) string {
	if t.IsZero() {
		t = time.Now()
	}
	return t.Format("2006-01-02 15:04:05 MST")
}
