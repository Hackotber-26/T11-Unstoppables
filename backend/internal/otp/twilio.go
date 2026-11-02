package otp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ErrNotConfigured = errors.New("SMS verification provider is not configured")
var ErrInvalidCode = errors.New("verification code is invalid or expired")
var ErrProviderRejected = errors.New("SMS provider rejected the verification request")

type TwilioVerify struct {
	accountSID string
	authToken  string
	serviceSID string
	client     *http.Client
	baseURL    string
}

func NewTwilioVerify(accountSID, authToken, serviceSID string) (*TwilioVerify, error) {
	configured := accountSID != "" || authToken != "" || serviceSID != ""
	if !configured {
		return nil, nil
	}
	if accountSID == "" || authToken == "" || serviceSID == "" {
		return nil, errors.New("all Twilio Verify credentials must be provided")
	}
	return &TwilioVerify{
		accountSID: accountSID,
		authToken:  authToken,
		serviceSID: serviceSID,
		client:     &http.Client{Timeout: 10 * time.Second},
		baseURL:    "https://verify.twilio.com/v2",
	}, nil
}

func (t *TwilioVerify) Start(ctx context.Context, phone string) error {
	if t == nil {
		return ErrNotConfigured
	}
	form := url.Values{"To": {phone}, "Channel": {"sms"}}
	_, err := t.post(ctx, "Verifications", form)
	return err
}

func (t *TwilioVerify) Check(ctx context.Context, phone, code string) error {
	if t == nil {
		return ErrNotConfigured
	}
	form := url.Values{"To": {phone}, "Code": {code}}
	payload, err := t.post(ctx, "VerificationCheck", form)
	if err != nil {
		if errors.Is(err, ErrProviderRejected) {
			return ErrInvalidCode
		}
		return err
	}
	var result struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(payload, &result); err != nil {
		return fmt.Errorf("decode SMS provider response: %w", err)
	}
	if result.Status != "approved" {
		return ErrInvalidCode
	}
	return nil
}

func (t *TwilioVerify) post(ctx context.Context, resource string, form url.Values) ([]byte, error) {
	endpoint := fmt.Sprintf("%s/Services/%s/%s", t.baseURL, url.PathEscape(t.serviceSID), resource)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("create SMS verification request: %w", err)
	}
	request.SetBasicAuth(t.accountSID, t.authToken)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := t.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("send SMS verification request: %w", err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 16*1024))
	if err != nil {
		return nil, fmt.Errorf("read SMS verification response: %w", err)
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		if response.StatusCode == http.StatusBadRequest || response.StatusCode == http.StatusNotFound {
			return nil, ErrProviderRejected
		}
		return nil, fmt.Errorf("SMS provider returned HTTP %d", response.StatusCode)
	}
	return payload, nil
}
