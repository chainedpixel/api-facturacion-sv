package contingency

// DeferredDocument is a DTE generated in contingency: the deferred structure delivered to the consumer and the JWS
// signed over that same structure, which is later transmitted unchanged in the contingency batch.
type DeferredDocument struct {
	Document       map[string]interface{}
	SignedDocument string
}

// Response returns the document delivered to the consumer: the signed structure plus its electronic signature.
func (d *DeferredDocument) Response() map[string]interface{} {
	response := make(map[string]interface{}, len(d.Document)+1)
	for key, value := range d.Document {
		response[key] = value
	}
	response["firmaElectronica"] = d.SignedDocument
	return response
}
