package scrape

import (
	"errors"
	"go-weaviate-deepseek/ext"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/microcosm-cc/bluemonday"
	"github.com/sirupsen/logrus"
)

func l() *logrus.Entry {
	return ext.LF("scrape")
}

type Scraper struct {
	EntryURL        string   `json:"entry_url"`
	ContinueDomains []string `json:"continue_domains"`

	depth int `json:"-"`

	// 下面两个必须分开保存，因为url可能会被重定向，最终爬取的结果可能是同一个URL
	handledURLs map[string]bool  `json:"-"`
	result      map[string]ext.M `json:"-"`
}

func NewScraper(entryURL string, continueDomains []string) *Scraper {
	if len(continueDomains) == 0 {
		continueDomains = []string{getRootDomain(entryURL)}
	}

	return &Scraper{
		EntryURL:        entryURL,
		ContinueDomains: continueDomains,

		handledURLs: make(map[string]bool),
		result:      make(map[string]ext.M),
	}
}

// SetDepth 0(default) | 1, 0: unlimit, 1: only scrape the entry url
func (s *Scraper) SetDepth(depth int) {
	s.depth = depth
}

func (s *Scraper) Start() (map[string]ext.M, error) {
	c := colly.NewCollector()
	c.SetRequestTimeout(10 * time.Second)
	// c.MaxDepth = 1
	sa := GetSanitizer()

	c.SetRedirectHandler(func(req *http.Request, via []*http.Request) error {
		if s.canContinue(req.URL.String()) {
			return nil
		}
		return newCanNotContinue(req.URL.String())
	})

	// c.Limits([]*colly.LimitRule{
	// 	{
	// 		DomainRegexp: `eggman.tv`,
	// 	},
	// })

	if s.depth == 0 {
		// Find and visit all links
		c.OnHTML("a[href]", func(e *colly.HTMLElement) {
			href := e.Attr("href")
			urlStr := buildURL(e.Request, href)
			if s.canContinue(urlStr) {
				// time.Sleep(2 * time.Second)
				c.Visit(urlStr)
			}
		})
	}

	c.OnError(func(r *colly.Response, err error) {
		url := r.Request.URL.String()
		if _, ok := err.(*errCanNotContinue); !ok {
			s.result[url] = ext.M{
				"status": "error",
				"text":   err.Error(),
			}
			l().Printf("err: %s, url: %s", err, r.Request.URL.String())
		}
	})

	c.OnRequest(func(r *colly.Request) {
		urlStr := r.URL.String()
		if s.canContinue(urlStr) {
			s.handledURLs[urlStr] = true
		} else {
			r.Abort()
		}
	})

	c.OnResponse(func(e *colly.Response) {
		url := e.Request.URL.String()
		url = buildURL(e.Request, url)
		if len(url) == 0 {
			return
		}

		if _, exists := s.result[url]; exists {
			return
		}
		// l().Printf("url: %s, body: %s\n", e.Request.URL.String(), string(e.Body))
		if shouldSkipMIMETypes(e.Headers.Get("Content-Type")) {
			// Skip the request if it's a binary file
			return
		}

		c := string(e.Body)
		c = sa.Sanitize(c)
		c = multiBlankRE.ReplaceAllString(c, " ")
		if multiBlankRE.ReplaceAllString(c, "") == "" {
			return
		}

		doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(e.Body)))
		if err != nil {
			l().Warn("parse body with goquery err:", err)
			return
		}
		title := doc.Find("title").Text()
		firstImgURL := doc.Find("img").First().AttrOr("href", "")
		s.result[url] = ext.M{
			"status":   "ok",
			"text":     c,
			"title":    title,
			"icon_url": firstImgURL,
		}
		if len(c) > 20 {
			l().Printf("done, url: %s, content: %s", url, c[0:20])
		} else {
			l().Printf("done, url: %s, content: %s", url, c)
		}
	})

	c.OnScraped(func(r *colly.Response) {
		// l().Println("one url done")
	})

	c.Visit(s.EntryURL)
	c.Wait()
	return s.result, nil
}

var multiBlankRE = regexp.MustCompile(`\s+`)

type errCanNotContinue struct {
	URL string
	Err error
}

func (ec *errCanNotContinue) Error() string {
	return ec.Err.Error()
}

func newCanNotContinue(urlStr string) *errCanNotContinue {
	return &errCanNotContinue{
		URL: urlStr,
		Err: errors.New("can not continue, url: " + urlStr),
	}
}

func buildURL(r *colly.Request, urlStr string) string {
	urlStr = strings.ToLower(urlStr)
	// return urlStr
	if strings.HasPrefix(urlStr, "https://") || strings.HasPrefix(urlStr, "http://") {
		return urlStr
	} else if strings.HasPrefix(urlStr, "mailto:") || strings.HasPrefix(urlStr, "#") {
		return ""
	} else if strings.HasPrefix(urlStr, "/") {
		return r.URL.Scheme + "://" + r.URL.Host + urlStr
	} else {
		// l().Println("unknown urlStr:", urlStr)
		// return r.URL.String() + urlStr
		return ""
	}
}

func (s *Scraper) canContinue(urlStr string) bool {
	if len(urlStr) == 0 {
		return false
	}
	if _, exists := s.handledURLs[urlStr]; !exists {
		if s.inContinueDomain(urlStr) && !shouldSkipSuffixes(urlStr) {
			return true
		}
	}
	return false
}

func (s *Scraper) inContinueDomain(urlStr string) bool {
	urlStr = strings.ToLower(urlStr)
	u, err := url.Parse(urlStr)
	if err != nil {
		l().Printf("parse url error: %s, url: %s", err, urlStr)
		return false
	}

	if strings.HasPrefix(urlStr, "https://") || strings.HasPrefix(urlStr, "http://") {
		for _, d := range s.ContinueDomains {
			if strings.Contains(u.Host, d) {
				return true
			}
		}
		return false
	}
	return true
}

func shouldSkipSuffixes(urlStr string) bool {
	for _, d := range skipURLStrings {
		if strings.Contains(urlStr, d) {
			return true
		}
	}
	return false
}

func shouldSkipMIMETypes(mt string) bool {
	for _, d := range skipMIMETypes {
		if strings.Contains(mt, d) {
			return true
		}
	}
	return false
}

func GetSanitizer() *bluemonday.Policy {
	// p := bluemonday.UGCPolicy()
	p := bluemonday.NewPolicy()
	p.AddSpaceWhenStrippingTag(true)
	return p
}

func getRootDomain(urlStr string) string {
	ul, _ := url.Parse(urlStr)
	s := strings.Split(ul.Hostname(), ".")
	return strings.Join(s[len(s)-2:], ".")
}
