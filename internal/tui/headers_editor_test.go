package tui

import "testing"

func TestParseHeaders(t *testing.T) {
	input := "X-Foo: bar\nX-Baz: qux\n\n"
	headers, err := parseHeaders(input)
	if err != nil {
		t.Fatal(err)
	}
	if headers["X-Foo"] != "bar" || headers["X-Baz"] != "qux" {
		t.Errorf("unexpected: %v", headers)
	}
}

func TestParseHeadersInvalid(t *testing.T) {
	_, err := parseHeaders("invalid-no-colon")
	if err == nil {
		t.Error("expected error for missing colon")
	}
}

func TestFormatHeaders(t *testing.T) {
	headers := map[string]string{"X-Foo": "bar", "X-Baz": "qux"}
	text := formatHeaders(headers)
	if len(text) == 0 {
		t.Error("expected non-empty output")
	}
}

func TestParseHeadersEmpty(t *testing.T) {
	headers, err := parseHeaders("")
	if err != nil {
		t.Fatal(err)
	}
	if len(headers) != 0 {
		t.Errorf("expected empty map, got: %v", headers)
	}
}

func TestParseHeadersValueWithColon(t *testing.T) {
	input := "Authorization: Bearer token:abc"
	headers, err := parseHeaders(input)
	if err != nil {
		t.Fatal(err)
	}
	if headers["Authorization"] != "Bearer token:abc" {
		t.Errorf("unexpected value: %q", headers["Authorization"])
	}
}

func TestFormatHeadersEmpty(t *testing.T) {
	text := formatHeaders(nil)
	if text != "" {
		t.Errorf("expected empty string, got: %q", text)
	}
}
