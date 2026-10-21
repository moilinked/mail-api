package mailer

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	gomail "github.com/wneessen/go-mail"
)

// ParseRecipient 校验单个收件人，并去掉显示名，只保留邮箱地址。
func ParseRecipient(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("recipient is required")
	}
	if strings.ContainsAny(raw, "\r\n") {
		return "", errors.New("invalid recipient")
	}

	addr, err := mail.ParseAddress(raw)
	if err != nil || addr.Address == "" || !strings.Contains(addr.Address, "@") {
		return "", errors.New("invalid recipient")
	}
	return addr.Address, nil
}

// SendHTML 发送一封 HTML 邮件，并附上纯文本版本。
func (m *Mailer) SendHTML(ctx context.Context, to, subject, htmlBody, textBody string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if strings.TrimSpace(to) == "" {
		return errors.New("recipient is required")
	}
	if strings.TrimSpace(subject) == "" {
		return errors.New("subject is required")
	}
	if strings.TrimSpace(htmlBody) == "" {
		return errors.New("html body is required")
	}

	msg := gomail.NewMsg()
	if err := msg.From(m.from); err != nil {
		return fmt.Errorf("set from: %w", err)
	}
	if err := msg.To(to); err != nil {
		return fmt.Errorf("set to: %w", err)
	}
	msg.Subject(subject)
	if strings.TrimSpace(textBody) != "" {
		msg.SetBodyString(gomail.TypeTextPlain, textBody)
		msg.AddAlternativeString(gomail.TypeTextHTML, htmlBody)
	} else {
		msg.SetBodyString(gomail.TypeTextHTML, htmlBody)
	}

	if err := m.send(ctx, msg); err != nil {
		return fmt.Errorf("send email: %w", err)
	}
	return nil
}

// SendTemplateFile 按 HTML 模板文件给一个收件人发信。
func SendTemplateFile(ctx context.Context, m *Mailer, path, to string) error {
	if m == nil {
		return errors.New("mailer is required")
	}
	recipient, err := ParseRecipient(to)
	if err != nil {
		return err
	}
	tmpl, err := LoadEmailTemplate(path)
	if err != nil {
		return err
	}
	return m.SendHTML(ctx, recipient, tmpl.Subject, tmpl.HTML, tmpl.Text)
}
