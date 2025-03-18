package scraping

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"

	"github.com/Vai3soh/lookingVps/internal/speedtester"
	"github.com/gocolly/colly/v2"
)

type SaveData struct {
	HostingName   string
	Country       string
	HostingURL    string
	SpeedTest     float64
	SpeedTestLink string
}

type ScrapeConfig struct {
	DownloadSize  int
	DownloadLimit int
	TimeoutSec    int
	MainPageURL   string
	LogOutputFile string
}

func NewScrapeConfig(downloadSize, downloadLimit, timeoutSec int, mainPageURL, logOutputFile string) *ScrapeConfig {
	return &ScrapeConfig{
		DownloadSize:  downloadSize,
		DownloadLimit: downloadLimit,
		TimeoutSec:    timeoutSec,
		MainPageURL:   mainPageURL,
		LogOutputFile: logOutputFile,
	}
}

type collectors struct {
	colly.Collector
	selectors     [5]string
	Debug         bool
	CollectedData []SaveData
	DataMutex     sync.Mutex
}

func NewCollectors(selectors [5]string, ua string, debug bool) *collectors {
	return &collectors{
		Collector:     *colly.NewCollector(colly.UserAgent(ua), colly.AllowURLRevisit()),
		selectors:     selectors,
		Debug:         debug,
		CollectedData: make([]SaveData, 0),
	}
}

func (c *collectors) Cloning() *colly.Collector {
	return c.Clone()
}

func (c *collectors) GetCollectedData() []SaveData {
	if c.Debug {
		log.Println("[DEBUG] Entering GetCollectedData")
	}
	c.DataMutex.Lock()
	defer func() {
		c.DataMutex.Unlock()
		if c.Debug {
			log.Println("[DEBUG] Exiting GetCollectedData")
		}
	}()

	result := make([]SaveData, len(c.CollectedData))
	copy(result, c.CollectedData)
	if c.Debug {
		log.Printf("[DEBUG] GetCollectedData returning %d records", len(result))
	}
	return result
}

func (c *collectors) fetchHostingDetails(detailPageURL string) (string, string, error) {
	cl := c.Clone()
	var hostingName, hostingLink string
	done := make(chan struct{})
	var once sync.Once

	closeDone := func() {
		once.Do(func() {
			close(done)
		})
	}

	cl.OnHTML("div.card-header.border-bottom.text-center", func(e *colly.HTMLElement) {
		hostingName = strings.TrimSpace(e.ChildText(".card-title"))
		hostingLink = strings.TrimSpace(e.ChildText("p.card-text a"))
		closeDone()
	})
	cl.OnError(func(r *colly.Response, err error) {
		closeDone()
	})

	if err := cl.Visit(detailPageURL); err != nil {
		return "", "", err
	}
	<-done

	if hostingName == "" {
		if c.Debug {
			log.Printf("[DEBUG] Hosting name not found on detail page %s", detailPageURL)
		}
		return "", "", errors.New("hosting name not found")
	}
	return hostingName, hostingLink, nil
}

func (c *collectors) fetchAlternateDownloadLink(detailPageURL string) (string, error) {
	cl := c.Clone()
	var altLink string
	done := make(chan struct{})

	cl.OnHTML("a", func(e *colly.HTMLElement) {
		href := e.Attr("href")
		if altLink == "" && strings.Contains(href, ".mb") {
			altLink = href
		}
	})
	cl.OnScraped(func(r *colly.Response) {
		close(done)
	})
	cl.OnError(func(r *colly.Response, err error) {
		close(done)
	})

	if err := cl.Visit(detailPageURL); err != nil {
		return "", err
	}
	<-done

	if altLink == "" {
		if c.Debug {
			log.Printf("[DEBUG] No alternate download link found on %s", detailPageURL)
		}
		return "", errors.New("no alternate download link found")
	}
	return altLink, nil
}

func (c *collectors) ScrapeServerData(ctx context.Context, cfg *ScrapeConfig) []SaveData {
	var allData []SaveData
	var mu sync.Mutex

	c.OnHTML("#LookingGlassServers", func(e *colly.HTMLElement) {
		select {
		case <-ctx.Done():
			return
		default:
		}
		if c.Debug {
			log.Printf("[DEBUG] Found LookingGlassServers container on %s", e.Request.URL)
		}
		e.ForEach("div.card.card-bordered.card-sm", func(i int, card *colly.HTMLElement) {
			select {
			case <-ctx.Done():
				return
			default:
			}

			data, err := c.processServerCard(card, ctx, cfg)
			if err != nil {
				if c.Debug {
					log.Printf("[DEBUG] Skipping card %d: %v", i, err)
				}
				return
			}

			mu.Lock()
			allData = append(allData, data)
			mu.Unlock()

			c.DataMutex.Lock()
			c.CollectedData = append(c.CollectedData, data)
			c.DataMutex.Unlock()

			if c.Debug {
				log.Printf("[DEBUG] Added SaveData: %+v", data)
			}
		})
	})

	c.OnHTML("#PaginationBottom-0", func(e *colly.HTMLElement) {
		c.handlePagination(e, ctx)
	})

	if c.Debug {
		log.Printf("[DEBUG] Starting scraping at: %s", cfg.MainPageURL)
	}
	c.Visit(cfg.MainPageURL)
	c.Wait()
	return allData
}

func (c *collectors) processServerCard(card *colly.HTMLElement, ctx context.Context, cfg *ScrapeConfig) (SaveData, error) {
	detailLink := card.ChildAttr("div.card-header a.link", "href")
	if detailLink == "" {
		return SaveData{}, fmt.Errorf("detail page link not found")
	}
	detailPageURL := card.Request.AbsoluteURL(detailLink)

	location := strings.TrimSpace(card.ChildText("div.location"))
	if location == "" {
		location = strings.TrimSpace(card.ChildText("div.card-header a.link"))
	}
	if location == "" {
		location = "Unknown"
	}

	hostingName, hostingWebsite, err := c.fetchHostingDetails(detailPageURL)
	if err != nil {
		return SaveData{}, fmt.Errorf("failed to fetch hosting details from %s: %w", detailPageURL, err)
	}

	expectedPattern := "/" + strconv.Itoa(cfg.DownloadSize) + ".mb"
	if c.Debug {
		log.Printf("[DEBUG] Expected test link pattern: %s", expectedPattern)
	}

	var expectedLink, fallbackLink string
	card.ForEach("a", func(_ int, a *colly.HTMLElement) {
		href := a.Attr("href")
		if href != "" {
			if fallbackLink == "" {
				fallbackLink = href
			}
			if expectedLink == "" && strings.Contains(href, expectedPattern) {
				expectedLink = href
			}
		}
	})

	var testLink string
	if expectedLink != "" {
		testLink = expectedLink
	} else {
		testLink = fallbackLink
		if c.Debug {
			log.Printf("[DEBUG] No link matching expected pattern found, using fallback link: %s", testLink)
		}
	}
	if testLink == "" {
		return SaveData{}, fmt.Errorf("no test link found in card")
	}
	if !strings.HasPrefix(testLink, "http") {
		testLink = card.Request.AbsoluteURL(testLink)
	}

	st := speedtester.NewSpeedTester(cfg.TimeoutSec, c.Debug)
	speed, err := st.PerformSpeedTest(ctx, testLink, cfg.DownloadLimit, cfg.LogOutputFile)
	if err != nil &&
		((errors.Is(err, context.DeadlineExceeded)) || strings.Contains(err.Error(), "Client.Timeout")) &&
		fallbackLink != "" && fallbackLink != testLink {
		fallbackAbs := fallbackLink
		if !strings.HasPrefix(fallbackAbs, "http") {
			fallbackAbs = card.Request.AbsoluteURL(fallbackAbs)
		}
		if strings.Contains(fallbackAbs, "/companies/") {
			if c.Debug {
				log.Printf("[DEBUG] Timeout for link %s. Trying fallback detail page: %s", testLink, fallbackAbs)
			}
			altLink, errAlt := c.fetchAlternateDownloadLink(fallbackAbs)
			if errAlt == nil && altLink != "" {
				if !strings.HasPrefix(altLink, "http") {
					altLink = card.Request.AbsoluteURL(altLink)
				}
				if c.Debug {
					log.Printf("[DEBUG] Alternate download link extracted: %s", altLink)
				}
				speed, err = st.PerformSpeedTest(ctx, altLink, cfg.DownloadLimit, cfg.LogOutputFile)
				if err != nil {
					if c.Debug {
						log.Printf("[DEBUG] Alternate speed test failed for %s: %v", altLink, err)
					}
					speed = 0.0
				} else {
					testLink = altLink
				}
			} else {
				if c.Debug {
					log.Printf("[DEBUG] Failed to fetch alternate download link from %s: %v", fallbackAbs, errAlt)
				}
				speed = 0.0
			}
		} else {
			if c.Debug {
				log.Printf("[DEBUG] Timeout for link %s. Trying fallback link directly: %s", testLink, fallbackAbs)
			}
			speed, err = st.PerformSpeedTest(ctx, fallbackAbs, cfg.DownloadLimit, cfg.LogOutputFile)
			if err != nil {
				if c.Debug {
					log.Printf("[DEBUG] Fallback link %s failed: %v", fallbackAbs, err)
				}
				speed = 0.0
			}
		}
	} else if err != nil {
		if c.Debug {
			log.Printf("[DEBUG] Speed test error for %s: %v", testLink, err)
		}
		speed = 0.0
	}

	data := SaveData{
		HostingName:   hostingName,
		Country:       location,
		HostingURL:    hostingWebsite,
		SpeedTest:     speed,
		SpeedTestLink: testLink,
	}
	return data, nil
}

func (c *collectors) handlePagination(e *colly.HTMLElement, ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	default:
	}
	if c.Debug {
		log.Printf("[DEBUG] Found pagination container on %s", e.Request.URL)
	}
	active := e.DOM.Find("ul.pagination li.page-item.active")
	if active.Length() > 0 {
		nextElem := active.Next()
		href, exists := nextElem.Find("a.page-link").Attr("href")
		if exists && href != "" {
			nextPageURL := e.Request.AbsoluteURL(href)
			if c.Debug {
				log.Printf("[DEBUG] Next page found: %s", nextPageURL)
			}
			select {
			case <-ctx.Done():
				return
			default:
			}
			e.Request.Visit(nextPageURL)
		} else {
			if c.Debug {
				log.Printf("[DEBUG] Next page link not found on %s", e.Request.URL)
			}
		}
	}
}
