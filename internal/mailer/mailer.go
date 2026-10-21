// Package mailer 通过飞书 SMTP 发送水质报告邮件。
package mailer

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/wneessen/go-mail"
)

const pdfContentType = mail.ContentType("application/pdf")

// Config 是当前选中的 SMTP 连接配置。
type Config struct {
	Provider   string
	Host       string
	Port       int
	Username   string
	Password   string
	Encryption string
}

// Report 是一封水质报告邮件。计数字段使用展示文本，和 JS 版 String() 后的结果一致。
type Report struct {
	To                     string
	FileName               string
	PDF                    []byte
	Zip                    string
	SystemName             string
	TotalAllCount          string
	TotalContaminantsCount string
}

// Mailer 发送水质报告邮件。
type Mailer struct {
	from string
	send func(ctx context.Context, msgs ...*mail.Msg) error
}

// New 按配置创建 SMTP 客户端。encryption 为 starttls 或 ssl，认证方式为 AUTH LOGIN。
func New(cfg Config) (*Mailer, error) {
	if cfg.Host == "" {
		return nil, errors.New("smtp host is required")
	}
	if cfg.Port == 0 {
		return nil, errors.New("smtp port is required")
	}
	if cfg.Username == "" || cfg.Password == "" {
		return nil, errors.New("smtp username and password are required")
	}

	options := []mail.Option{
		mail.WithPort(cfg.Port),
		mail.WithSMTPAuth(mail.SMTPAuthLogin),
		mail.WithUsername(cfg.Username),
		mail.WithPassword(cfg.Password),
		mail.WithTimeout(30 * time.Second),
	}
	switch cfg.Encryption {
	case encryptionSTARTTLS:
		options = append(options, mail.WithTLSPolicy(mail.TLSMandatory))
	case encryptionSSL:
		options = append(options, mail.WithSSL())
	default:
		return nil, fmt.Errorf("unsupported smtp encryption %q", cfg.Encryption)
	}

	client, err := mail.NewClient(cfg.Host, options...)
	if err != nil {
		return nil, fmt.Errorf("create smtp client: %w", err)
	}

	return &Mailer{
		from: cfg.Username,
		send: client.DialAndSendWithContext,
	}, nil
}

// SendReport 把水质报告 PDF 作为附件发送。
func (m *Mailer) SendReport(ctx context.Context, report Report) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(report.To) == "" {
		return errors.New("recipient is required")
	}
	if strings.TrimSpace(report.FileName) == "" {
		return errors.New("attachment file name is required")
	}
	if len(report.PDF) == 0 {
		return errors.New("pdf attachment is required")
	}

	htmlBody, err := buildReportEmailHTML(report)
	if err != nil {
		return fmt.Errorf("build html body: %w", err)
	}

	msg := mail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(report.To); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	msg.Subject(subject)
	msg.SetBodyString(mail.TypeTextPlain, buildReportText(report))
	msg.AddAlternativeString(mail.TypeTextHTML, htmlBody)
	msg.AttachReadSeeker(report.FileName, bytes.NewReader(report.PDF), mail.WithFileContentType(pdfContentType))

	if err := m.send(ctx, msg); err != nil {
		return fmt.Errorf("send report email: %w", err)
	}
	return nil
}
