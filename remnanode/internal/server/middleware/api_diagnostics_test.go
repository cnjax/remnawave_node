package middleware

import (
	"strings"
	"testing"
)

func TestShouldLogAPIDiagnostic(t *testing.T) {
	if !shouldLogAPIDiagnostic(500, `{"message":"failed"}`) {
		t.Fatal("expected HTTP 500 to be logged")
	}
	if !shouldLogAPIDiagnostic(200, `{"response":{"isStarted":false,"error":"failed"}}`) {
		t.Fatal("expected failed xray start response to be logged")
	}
	if !shouldLogAPIDiagnostic(200, `{"response":{"success":false,"error":"failed"}}`) {
		t.Fatal("expected failed handler response to be logged")
	}
	if shouldLogAPIDiagnostic(200, `{"response":{"success":true,"error":null}}`) {
		t.Fatal("did not expect successful response to be logged")
	}
}

func TestSanitizeDiagnosticBody(t *testing.T) {
	body := `{"password":"p","uuid":"u","token":"t","safe":"ok"}`
	got := sanitizeDiagnosticBody(body)

	for _, secret := range []string{"p", "u", "t"} {
		if strings.Contains(got, `"`+secret+`"`) {
			t.Fatalf("secret %q was not redacted: %s", secret, got)
		}
	}
	if !strings.Contains(got, `"safe":"ok"`) {
		t.Fatalf("non-sensitive field was unexpectedly changed: %s", got)
	}
}

func TestLimitedBuffer(t *testing.T) {
	buf := &limitedBuffer{limit: 4}
	n, err := buf.Write([]byte("abcdef"))
	if err != nil {
		t.Fatal(err)
	}
	if n != 6 {
		t.Fatalf("Write must report original length, got %d", n)
	}
	if got := buf.String(); got != "abcd" {
		t.Fatalf("unexpected captured body: %q", got)
	}
	if !buf.Truncated() {
		t.Fatal("expected buffer to be marked truncated")
	}
}
