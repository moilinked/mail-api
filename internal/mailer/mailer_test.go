package mailer

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/wneessen/go-mail"
)

func TestBuildReportEmailHTML(t *testing.T) {
	tests := []struct {
		name   string
		report Report
		want   []string
		reject []string
	}{
		{
			name: "escapes dynamic text and keeps zip link",
			report: Report{
				Zip:                    `10001"><script>`,
				SystemName:             `City <Water>`,
				TotalAllCount:          "42",
				TotalContaminantsCount: "7",
			},
			want: []string{
				"10001&#34;&gt;&lt;script&gt;",
				"City &lt;Water&gt;",
				">42<",
				">7<",
				"https://www.waterdropfilter.com/pages/test-your-tap-water?zip=10001%22%3E%3Cscript%3E",
				"Privacy policy",
				"service@waterdropfilter.com",
			},
			reject: []string{"<script>"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildReportEmailHTML(tt.report)
			if err != nil {
				t.Fatalf("build html: %v", err)
			}
			for _, snippet := range tt.want {
				if !strings.Contains(got, snippet) {
					t.Fatalf("html missing %q", snippet)
				}
			}
			for _, snippet := range tt.reject {
				if strings.Contains(got, snippet) {
					t.Fatalf("html contains %q", snippet)
				}
			}
		})
	}
}

func TestBuildReportText(t *testing.T) {
	got := buildReportText(Report{
		Zip:                    "10001",
		SystemName:             "Example Water",
		TotalAllCount:          "42",
		TotalContaminantsCount: "7",
	})

	want := "Your personalized water quality report is ready.\n\n" +
		"ZIP Code: 10001\n" +
		"Water Providers: Example Water\n" +
		"42 Total Contaminants detected.\n" +
		"7 Contaminants Exceed EWG's Health Guidelines.\n\n" +
		"We've attached your personalized PDF report."
	if got != want {
		t.Fatalf("text = %q, want %q", got, want)
	}
}

func TestSendReport(t *testing.T) {
	tests := []struct {
		name    string
		report  Report
		sendErr error
		wantErr string
	}{
		{
			name:    "requires recipient",
			report:  Report{FileName: "report.pdf", PDF: []byte("%PDF")},
			wantErr: "recipient is required",
		},
		{
			name:    "requires file name",
			report:  Report{To: "user@example.com", PDF: []byte("%PDF")},
			wantErr: "attachment file name is required",
		},
		{
			name:    "requires pdf",
			report:  Report{To: "user@example.com", FileName: "report.pdf"},
			wantErr: "pdf attachment is required",
		},
		{
			name: "wraps smtp error",
			report: Report{
				To:       "user@example.com",
				FileName: "report.pdf",
				PDF:      []byte("%PDF"),
			},
			sendErr: errors.New("smtp down"),
			wantErr: "send report email: smtp down",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := &Mailer{
				from: "info@waterdropfilter.com",
				send: func(context.Context, ...*mail.Msg) error {
					return tt.sendErr
				},
			}
			err := sender.SendReport(context.Background(), tt.report)
			if err == nil || err.Error() != tt.wantErr {
				t.Fatalf("error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}

func TestSendReportComposesMessage(t *testing.T) {
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

	err := sender.SendReport(context.Background(), Report{
		To:                     "user@example.com",
		FileName:               "ewg-report.pdf",
		PDF:                    []byte("%PDF-1.4"),
		Zip:                    "10001",
		SystemName:             "Example Water",
		TotalAllCount:          "42",
		TotalContaminantsCount: "7",
	})
	if err != nil {
		t.Fatalf("send: %v", err)
	}

	got := raw.String()
	for _, snippet := range []string{
		"Subject: EWG Water Quality Report",
		"info@waterdropfilter.com",
		"user@example.com",
		"ewg-report.pdf",
		"application/pdf",
		"ZIP Code: 10001",
		">42<",
	} {
		if !strings.Contains(got, snippet) {
			t.Fatalf("message missing %q", snippet)
		}
	}
}
