package mailer

import (
	"html/template"
	"net/url"
	"strings"
)

const (
	logoURL          = "https://cdn.shopify.com/s/files/1/0078/6156/7570/files/waterdrop-logo-new.png?v=1701935920"
	siteWaterTestURL = "https://www.waterdropfilter.com/pages/test-your-tap-water"
	privacyURL       = "https://www.waterdropfilter.com/policies/privacy-policy"
	termsURL         = "https://www.waterdropfilter.com/policies/terms-of-service"
	subject          = "EWG Water Quality Report"
)

type reportView struct {
	Zip                    string
	SystemName             string
	TotalAllCount          string
	TotalContaminantsCount string
	SiteURL                template.URL
}

var reportHTMLTmpl = template.Must(template.New("report").Parse(`<!DOCTYPE html><html lang="en"><head><meta charset="UTF-8" /><meta name="viewport" content="width=device-width, initial-scale=1.0" /><title>EWG Water Quality Report</title></head><body style="margin:0;padding:0;background:#ffffff;font-family:Arial,Helvetica,sans-serif;color:#222222;"><table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background:#ffffff;"><tr><td align="center" style="padding:32px 16px;"><table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;"><tr><td align="center" style="padding-bottom:28px;"><img src="` + logoURL + `" alt="Waterdrop" width="220" style="display:block;border:0;max-width:220px;height:auto;" /></td></tr><tr><td style="font-size:22px;line-height:1.4;font-weight:bold;padding-bottom:28px;">Your personalized water quality report is ready.</td></tr><tr><td style="padding-bottom:18px;"><div style="font-size:14px;line-height:1.4;color:#222222;">ZIP Code</div><div style="font-size:16px;line-height:1.5;font-weight:bold;color:#222222;">{{.Zip}}</div></td></tr><tr><td style="padding-bottom:22px;"><div style="font-size:14px;line-height:1.4;color:#222222;">Water Providers</div><div style="font-size:16px;line-height:1.5;font-weight:bold;color:#222222;">{{.SystemName}}</div></td></tr><tr><td style="font-size:16px;line-height:1.6;padding-bottom:8px;"><span style="color:#c9972e;font-weight:bold;">{{.TotalAllCount}}</span> Total Contaminants detected.</td></tr><tr><td style="font-size:16px;line-height:1.6;padding-bottom:24px;"><span style="color:#d42d28;font-weight:bold;">{{.TotalContaminantsCount}}</span> Contaminants Exceed EWG's Health Guidelines.</td></tr><tr><td style="font-size:15px;line-height:1.7;padding-bottom:18px;">We've attached your personalized PDF report, including your local water profile, key contaminants detected, exceedance levels based on EWG reference values, and recommended Waterdrop filtration options.</td></tr><tr><td style="font-size:15px;line-height:1.7;padding-bottom:36px;">View or download the attached report, or <a href="{{.SiteURL}}" style="color:#1a73e8;text-decoration:underline;">return to our website</a> to check your water quality again.</td></tr><tr><td align="center" style="font-size:16px;font-weight:bold;padding-bottom:12px;">Waterdrop</td></tr><tr><td align="center" style="font-size:12px;line-height:1.6;color:#666666;padding-bottom:16px;">You are receiving this email because you opted in on our website. Please do not reply to this email, if you have any questions, contact us at <a href="mailto:service@waterdropfilter.com" style="color:#666666;text-decoration:underline;">service@waterdropfilter.com</a>.</td></tr><tr><td align="center" style="font-size:12px;line-height:1.6;color:#666666;"><a href="` + privacyURL + `" style="color:#666666;text-decoration:underline;">Privacy policy</a>&nbsp;|&nbsp;<a href="` + termsURL + `" style="color:#666666;text-decoration:underline;">Terms of service</a></td></tr></table></td></tr></table></body></html>`))

func buildReportEmailHTML(report Report) (string, error) {
	var buf strings.Builder
	view := reportView{
		Zip:                    report.Zip,
		SystemName:             report.SystemName,
		TotalAllCount:          report.TotalAllCount,
		TotalContaminantsCount: report.TotalContaminantsCount,
		SiteURL:                template.URL(waterTestURL(report.Zip)),
	}
	if err := reportHTMLTmpl.Execute(&buf, view); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func buildReportText(report Report) string {
	return "Your personalized water quality report is ready.\n\n" +
		"ZIP Code: " + report.Zip + "\n" +
		"Water Providers: " + report.SystemName + "\n" +
		report.TotalAllCount + " Total Contaminants detected.\n" +
		report.TotalContaminantsCount + " Contaminants Exceed EWG's Health Guidelines.\n\n" +
		"We've attached your personalized PDF report."
}

func waterTestURL(zip string) string {
	return siteWaterTestURL + "?zip=" + encodeURIComponent(zip)
}

func encodeURIComponent(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}
