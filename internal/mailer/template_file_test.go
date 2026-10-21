package mailer

import (
	"bytes"
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wneessen/go-mail"
)

func TestLoadVoucherTemplate(t *testing.T) {
	got, err := LoadEmailTemplate(filepath.Join("..", "..", "email-template.html"))
	if err != nil {
		t.Fatalf("load template: %v", err)
	}
	if got.Subject != "Your Black Friday Voucher Is Here!" {
		t.Fatalf("subject = %q", got.Subject)
	}

	for _, snippet := range []string{
		"Welcome to Waterdrop",
		"Code: 8G5F AGEG HED9 EA63",
		"Valid time: 09/09/2026 ~ 10/08/2026",
		"Shop Now https://www.waterdropfilter.com",
		"service@waterdropfilter.com",
	} {
		if !strings.Contains(got.Text, snippet) {
			t.Fatalf("text missing %q\n%s", snippet, got.Text)
		}
	}
	if strings.Contains(got.Text, "Your reward is ready") {
		t.Fatalf("preheader leaked into text:\n%s", got.Text)
	}
	if !strings.Contains(got.HTML, "8G5F AGEG HED9 EA63") {
		t.Fatal("html missing coupon code")
	}
}

func TestLoadEmailTemplateMissing(t *testing.T) {
	_, err := LoadEmailTemplate(filepath.Join(t.TempDir(), "missing.html"))
	if !errors.Is(err, ErrTemplate) {
		t.Fatalf("error = %v, want ErrTemplate", err)
	}
}

func TestParseRecipient(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    string
		wantErr string
	}{
		{name: "plain", raw: "user@example.com", want: "user@example.com"},
		{name: "named", raw: "User <user@example.com>", want: "user@example.com"},
		{name: "empty", raw: "  ", wantErr: "recipient is required"},
		{name: "invalid", raw: "not-an-email", wantErr: "invalid recipient"},
		{name: "newline", raw: "user@example.com\nBcc: x", wantErr: "invalid recipient"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseRecipient(tt.raw)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if got != tt.want {
				t.Fatalf("address = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSendTemplateFileComposesMessage(t *testing.T) {
	var raw bytes.Buffer
	sender := &Mailer{
		from: "info@waterdropfilter.com",
		send: func(_ context.Context, msgs ...*mail.Msg) error {
			if len(msgs) != 1 {
				return errors.New("unexpected message count")
			}
			_, err := msgs[0].WriteTo(&raw)
			return err
		},
	}

	err := SendTemplateFile(
		context.Background(),
		sender,
		filepath.Join("..", "..", "email-template.html"),
		"User <user@example.com>",
	)
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	got := raw.String()
	for _, snippet := range []string{
		"Subject: Your Black Friday Voucher Is Here!",
		"info@waterdropfilter.com",
		"user@example.com",
		"text/html",
		"8G5F AGEG HED9 EA63",
		"Shop Now",
	} {
		if !strings.Contains(got, snippet) {
			t.Fatalf("message missing %q", snippet)
		}
	}
}
