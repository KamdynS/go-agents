package llm

import "testing"

func TestStructuredValidationAndSchema(t *testing.T) {
	// Sentiment schema has enum and bounds
	s := Sentiment{Sentiment: "positive", Score: 0.5}
	if err := s.Validate(); err != nil {
		t.Fatalf("validate: %v", err)
	}
	sch := s.JSONSchema()
	if sch["type"].(string) != "object" {
		t.Fatalf("bad schema")
	}

	// ParseStructured with pointer type
	jsonStr := `{"sentiment":"neutral","score":0.1}`
	resp, err := ParseStructured(jsonStr, &Sentiment{})
	if err != nil || !resp.Validation.Valid {
		t.Fatalf("parse: %v %#v", err, resp)
	}

	// Invalid case
	if _, err := ParseStructured(`{"sentiment":"unknown","score":0}`, Sentiment{}); err == nil {
		t.Fatalf("expected validation error")
	}
}
