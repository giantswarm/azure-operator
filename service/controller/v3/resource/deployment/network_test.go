package deployment

import "testing"

func Test_secondHalf(t *testing.T) {
	v, err := secondHalf("10.1.0.0/16")
	expected := "10.1.128.0/17"

	if err != nil {
		t.Errorf("unexpected error = %#v", err)
	}

	if v != expected {
		t.Errorf("expected cidr = %q, got %q", expected, v)
	}
}
