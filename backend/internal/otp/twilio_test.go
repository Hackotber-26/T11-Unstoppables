package otp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTwilioVerifyStartUsesPhoneAndSMSChannel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != "AC-test" || password != "secret" {
			t.Errorf("unexpected Twilio authentication")
		}
		if request.URL.Path != "/v2/Services/VA-test/Verifications" {
			t.Errorf("unexpected request path: %s", request.URL.Path)
		}
		if err := request.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if request.Form.Get("To") != "+919876543210" || request.Form.Get("Channel") != "sms" {
			t.Errorf("unexpected verification request form: %v", request.Form)
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte(`{"status":"pending"}`))
	}))
	defer server.Close()
	provider, err := NewTwilioVerify("AC-test", "secret", "VA-test")
	if err != nil {
		t.Fatal(err)
	}
	provider.baseURL = server.URL + "/v2"
	if err := provider.Start(context.Background(), "+919876543210"); err != nil {
		t.Fatal(err)
	}
}

func TestTwilioVerifyCheckRequiresApprovedStatus(t *testing.T) {
	tests := []struct {
		name       string
		status     string
		wantErr    error
	}{
		{name: "approved", status: "approved"},
		{name: "pending", status: "pending", wantErr: ErrInvalidCode},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				if err := request.ParseForm(); err != nil {
					t.Errorf("parse form: %v", err)
				}
				if request.Form.Get("Code") != "123456" {
					t.Errorf("unexpected code: %q", request.Form.Get("Code"))
				}
				_ = json.NewEncoder(writer).Encode(map[string]string{"status": testCase.status})
			}))
			defer server.Close()
			provider, err := NewTwilioVerify("AC-test", "secret", "VA-test")
			if err != nil {
				t.Fatal(err)
			}
			provider.baseURL = server.URL + "/v2"
			err = provider.Check(context.Background(), "+919876543210", "123456")
			if !errors.Is(err, testCase.wantErr) {
				t.Fatalf("Check() error = %v, want %v", err, testCase.wantErr)
			}
		})
	}
}
