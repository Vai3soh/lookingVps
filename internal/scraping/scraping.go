package scraping

import (
	"log"
	"strconv"
	"strings"

	"github.com/Vai3soh/speedtestVps/pkg/wget-go"
	"github.com/gocolly/colly/v2"
)

type HostingLink string

type network struct {
	PortSpeed    string
	LinkDownload string
	SpeedTest    float64
}

type SaveData struct {
	HostingLink
	DataCenterAndSpeedTestLink []map[string][]network
}

type collectors struct {
	colly.Collector
	network
	selectors [5]string
}

func NewNetwork(ps, link string, st float64) *network {
	return &network{
		PortSpeed:    ps,
		LinkDownload: link,
		SpeedTest:    st,
	}
}

func NewCollectors(s [5]string, ua string) *collectors {
	return &collectors{
		Collector: *colly.NewCollector(colly.UserAgent(ua)),
		selectors: s,
	}
}

func (c *collectors) Cloning() *colly.Collector {
	return c.Clone()
}

func sliceMode(items []string, item string) []string {
	newitems := []string{}

	for _, i := range items {
		if i != item || strings.Contains(item, "\n\t\t\t\t\t\t\t\t\t") {
			replacer := strings.NewReplacer("\n\t\t\t\t\t\t\t\t\t", " => ")
			i = replacer.Replace(strings.TrimSpace(i))
			newitems = append(newitems, i)
		}
	}
	return newitems
}

func (c *collectors) choiseLink(downloadFileSize int) bool {

	switch downloadFileSize {
	case 10:
		if strings.Contains(c.LinkDownload, "10.mb") {
			return true
		}
	case 100:
		if strings.Contains(c.LinkDownload, "100.mb") {
			return true
		}
	case 1000:
		if strings.Contains(c.LinkDownload, "1000.mb") {
			return true
		}
	}
	return false
}

func (c *collectors) speedTest(percentLimit, fileSave string) ([]float64, error) {
	spd, _, err := wget.WgetCli([]string{"", "-O", fileSave, "-L", percentLimit, c.LinkDownload})
	if err != nil {
		return nil, err
	}
	return spd, err
}

func (c *collectors) GetBufferSize(sl *[]chan []SaveData, globalUrl string) {

	c.OnHTML(c.selectors[0], func(e *colly.HTMLElement) {
		selector := "body > footer > div > div > div > div:nth-child(6) > a:nth-child(2)"
		s := strings.Split(e.ChildText(selector), " ")[0]
		n, _ := strconv.Atoi(s)

		closerChan := make(chan []SaveData, n*3)
		*sl = append(*sl, closerChan)

	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.Visit(globalUrl)
}

func (c *collectors) visitingUrls(selector string, detail *colly.Collector) {

	c.OnHTML(selector, func(e *colly.HTMLElement) {
		if strings.Contains(e.Attr("href"), "company") {
			link := e.Request.AbsoluteURL(e.Attr("href"))
			if visited, _ := detail.HasVisited(link); !visited {
				err := detail.Visit(link)
				if err != nil {
					log.Println("err:", err)
				}
			}
		}
	})

	detail.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})
}

func (c *collectors) AddNetworkToSlises(
	merge *[][]network, fileSize int,
	fileSave, percentLimit string, linkDownloads, titles []string,
) {

	group := []network{}
	for i := range linkDownloads {
		spd := 0.0
		nw := NewNetwork(titles[i], linkDownloads[i], spd)
		c.network = *nw
		ok := c.choiseLink(fileSize)

		if ok {

			spds, err := c.speedTest(percentLimit, fileSave)

			if err != nil {
				log.Println()
				log.Println(err)
			} else {
				if len(spds) != 0 {
					spd = spds[0]
					nw.SpeedTest = spd
				}
			}
		}

		group = append(group, *nw)

		if len(group) == 3 {
			*merge = append(*merge, group)
			group = []network{}
		}
	}

}

func (c *collectors) ReadDetailColl(fileSize int, percentLimit, globalUrl, fileSave string, closerChan chan []SaveData) []SaveData {

	s := &SaveData{}
	all := []SaveData{}
	detail := c.Cloning()

	c.visitingUrls(c.selectors[1], detail)

	detail.OnHTML(c.selectors[0], func(e *colly.HTMLElement) {

		merge := [][]network{}

		hostingLink := e.ChildText(c.selectors[2])
		dataCenters := e.ChildTexts(c.selectors[3])
		dataCenters = sliceMode(dataCenters, "Full info")
		linkDownloads := e.ChildAttrs(c.selectors[4], "href")
		titles := e.ChildAttrs(c.selectors[4], "title")

		c.AddNetworkToSlises(&merge, fileSize,
			fileSave, percentLimit,
			linkDownloads, titles,
		)

		s.HostingLink = HostingLink(hostingLink)
		sliceMaps := []map[string][]network{}
		if len(dataCenters) == len(merge) {
			for i := range dataCenters {
				sliceMaps = append(sliceMaps, map[string][]network{
					dataCenters[i]: merge[i],
				})
			}
		}
		s.DataCenterAndSpeedTestLink = sliceMaps
		all = append(all, *s)
		go func() {
			closerChan <- all
		}()
	})

	c.OnError(func(r *colly.Response, err error) {
		log.Println("Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	c.Visit(globalUrl)

	return all
}
