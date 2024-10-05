package licensedescriber

import (
	"fmt"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly"
	colly_debug "github.com/gocolly/colly/debug"
)

const tldrLegalDomain = "www.tldrlegal.com"

var tldrLegalLicenseNameMap = map[string]string{
	"apache-2.0": "apache-license-2-0-apache-2-0",

	// TODO: verify
	"mit":          "mit-license",
	"mpl-2.0":      "mpl-license-2-0-mozilla-public-license-2-0",
	"gpl-2.0":      "gpl-license-2-0-gnu-general-public-license-v2",
	"gpl-3.0":      "gpl-license-3-0-gnu-general-public-license-v3",
	"lgpl-2.1":     "lgpl-license-2-1-gnu-library-general-public-license-v2-1",
	"lgpl-3.0":     "lgpl-license-3-0-gnu-library-general-public-license-v3",
	"cc-by-4.0":    "cc-license-4-0-creative-commons-attribution-4-0-international",
	"cc-by-sa-4.0": "cc-license-4-0-creative-commons-attribution-share-alike-4-0-international",
	"unlicense":    "unlicense",
	"cc0-1.0":      "cc-license-0-0-creative-commons-public-domain-dedication",
	"ms-pl":        "ms-license-2-0-microsoft-public-license",
	"ms-rl":        "ms-license-2-0-microsoft-reciprocal-license",
	"cddl-1.0":     "cddl-license-1-0-common-development-and-distribution-license",
	"epl-2.0":      "epl-license-2-0-eclipse-public-license-2-0",
}

type LicenseSummary struct {
	Summary string

	// TODO: use
	Can    []string
	Must   []string
	Cannot []string
}

type TLDRLegalLicenseDescriber struct {
	urlPattern string
	collyObj   *colly.Collector
}

func NewTLDRLegalDescriber() *TLDRLegalLicenseDescriber {
	return &TLDRLegalLicenseDescriber{
		urlPattern: fmt.Sprintf("https://%s/license/%%s", tldrLegalDomain),
		collyObj: colly.NewCollector(
			colly.AllowedDomains(tldrLegalDomain),
			colly.MaxDepth(1),
			colly.Debugger(&slogDebugger{}),
		),
	}
}

func (d *TLDRLegalLicenseDescriber) Describe(license string) (string, error) {
	summary, err := d.GetLicenseSummary(license)
	if err != nil {
		return "", fmt.Errorf("failed to fetch license summary: %w", err)
	}

	return summary.Summary, nil

}

func (d *TLDRLegalLicenseDescriber) GetLicenseSummary(license string) (*LicenseSummary, error) {
	mappedLicenseName, ok := tldrLegalLicenseNameMap[strings.ToLower(license)]
	if !ok {
		// fall back to the license name as is
		mappedLicenseName = license
	}

	url := fmt.Sprintf(d.urlPattern, mappedLicenseName)

	logger := slog.With(
		"license", license,
		"mapped-license", mappedLicenseName,
		"url", url,
	)
	logger.Info("fetching license summary")

	var summary LicenseSummary
	d.collyObj.OnHTML("div[data-w-tab=\"Tab 1\"]", func(e *colly.HTMLElement) {
		summary.Summary = e.ChildText(".c-rich-text")
	})

	d.collyObj.OnHTML(".c-feature", func(e *colly.HTMLElement) {
		header := e.ChildText(".c-feature_header")

		var items []string
		e.ForEach(".c-feature_item", func(_ int, el *colly.HTMLElement) {
			items = append(items, el.ChildText(".cc-semibold"))
		})

		switch header {
		case "Can":
			summary.Can = items
		case "Must":
			summary.Must = items
		case "Cannot":
			summary.Cannot = items
		default:
			logger.Warn("unknown header encounted in GetLicenseSummary", "header", header)
		}
	})

	d.collyObj.OnError(func(r *colly.Response, err error) {
		slog.Error(
			"colly error encountered",
			"error", err,
			"url", r.Request.URL,
			"initiator-url", url,
			"license", license,
			"mapped-license", mappedLicenseName,
		)
	})

	err := d.collyObj.Visit(url)
	if err != nil {
		return nil, fmt.Errorf("failed to visit %s: %w", url, err)
	}
	return &summary, nil
}

func debugElement(s string, e *colly.HTMLElement) {
	h, err := e.DOM.Html()
	slog.Debug(s, "html", h, "err", err)
}

func debugSelection(s string, sel *goquery.Selection) {
	h, err := sel.Html()
	slog.Debug(s, "html", h, "err", err)
}

type slogDebugger struct {
	counter int32
	start   time.Time
}

func (d *slogDebugger) Init() error {
	d.counter = 0
	d.start = time.Now()
	return nil
}

func (d *slogDebugger) Event(e *colly_debug.Event) {
	i := atomic.AddInt32(&d.counter, 1)
	slog.Debug(
		"colly_event received",
		"counter", i,
		"collector_id", e.CollectorID,
		"request_id", e.RequestID,
		"type", e.Type,
		"values", e.Values,
		"duration", time.Since(d.start),
	)
}
