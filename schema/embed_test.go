package schema

import "testing"

func TestConfigSchemaCompiles(t *testing.T) {
	sch, err := ConfigSchema()
	if err != nil {
		t.Fatalf("failed to compile config schema: %v", err)
	}
	if sch == nil {
		t.Fatal("compiled schema is nil")
	}
}

func TestConfigSchemaIdempotent(t *testing.T) {
	sch1, err1 := ConfigSchema()
	sch2, err2 := ConfigSchema()
	if err1 != nil || err2 != nil {
		t.Fatalf("errors: %v, %v", err1, err2)
	}
	if sch1 != sch2 {
		t.Error("expected same schema instance from sync.Once")
	}
}
