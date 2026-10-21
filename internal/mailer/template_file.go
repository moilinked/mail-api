package mailer

import (
	"errors"
	"fmt"
	"html"
	"os"
	"regexp"
	"strings"
)

// ErrTemplate 表示 HTML 模板文件无法读取或内容不完整。
var ErrTemplate = errors.New("email template unavailable")

// EmailTemplate 是一封可直接发送的 HTML 邮件。
type EmailTemplate struct {
	Subject string
	HTML    string
	Text    string
}

var (
	titlePattern   = regexp.MustCompile(`(?is)<title>\s*(.*?)\s*</title>`)
	headPattern    = regexp.MustCompile(`(?is)<head[^>]*>.*?</head>`)
	commentPattern = regexp.MustCompile(`(?is)<!--.*?-->`)
	hiddenPattern  = regexp.MustCompile(`(?is)<div[^>]*style="[^"]*display:\s*none[^"]*"[^>]*>.*?</div>`)
	linkPattern    = regexp.MustCompile(`(?is)<a\b[^>]*href="([^"]+)"[^>]*>(.*?)</a>`)
	blockPattern   = regexp.MustCompile(`(?is)</(p|div|tr|h1|h2|h3|li|table)>`)
	brPattern      = regexp.MustCompile(`(?is)<br\s*/?>`)
	tagPattern     = regexp.MustCompile(`(?is)<[^>]+>`)
	spacePattern   = regexp.MustCompile(`[ \t]+`)
)

// LoadEmailTemplate 读取 HTML 文件。主题取 <title>，纯文本从可见内容生成。
func LoadEmailTemplate(path string) (EmailTemplate, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return EmailTemplate{}, fmt.Errorf("%w: %s", ErrTemplate, err.Error())
	}

	source := string(raw)
	subject := ""
	if match := titlePattern.FindStringSubmatch(source); len(match) == 2 {
		subject = strings.TrimSpace(html.UnescapeString(match[1]))
	}
	if subject == "" {
		return EmailTemplate{}, fmt.Errorf("%w: subject is empty", ErrTemplate)
	}
	if strings.TrimSpace(source) == "" {
		return EmailTemplate{}, fmt.Errorf("%w: html is empty", ErrTemplate)
	}

	return EmailTemplate{
		Subject: subject,
		HTML:    source,
		Text:    htmlToText(source),
	}, nil
}

func htmlToText(src string) string {
	s := headPattern.ReplaceAllString(src, "")
	s = commentPattern.ReplaceAllString(s, "")
	s = hiddenPattern.ReplaceAllString(s, "")
	s = linkPattern.ReplaceAllStringFunc(s, func(match string) string {
		parts := linkPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		href := strings.TrimSpace(html.UnescapeString(parts[1]))
		label := strings.TrimSpace(html.UnescapeString(tagPattern.ReplaceAllString(parts[2], "")))
		if label == "" {
			return href
		}
		if strings.EqualFold(label, href) || strings.EqualFold("mailto:"+label, href) {
			return label
		}
		return label + " " + href
	})
	const lineBreak = "\u0000"
	s = brPattern.ReplaceAllString(s, lineBreak)
	s = blockPattern.ReplaceAllString(s, lineBreak)
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\r", " ").Replace(s)
	s = tagPattern.ReplaceAllString(s, "")
	s = strings.ReplaceAll(s, lineBreak, "\n")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\u00a0", " ")

	var b strings.Builder
	blank := 0
	for line := range strings.SplitSeq(s, "\n") {
		line = strings.TrimSpace(spacePattern.ReplaceAllString(line, " "))
		if line == "" {
			blank++
			if blank > 1 {
				continue
			}
			if b.Len() > 0 {
				b.WriteByte('\n')
			}
			continue
		}
		blank = 0
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return strings.TrimSpace(b.String())
}
