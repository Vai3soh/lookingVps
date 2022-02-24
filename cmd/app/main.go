package main

import (
	"flag"
	"log"
	"strings"

	"github.com/Vai3soh/speedtestVps/pkg/wget-go"
	"github.com/gocolly/colly/v2"
)

type Country string
type HostingLink string
type City string
type Location string

type Data struct {
	Hosting map[HostingLink]map[Location][]network
}

type network struct {
	portSpeed    string
	linkDownload string
}

func removeAndModify(items []string, item string) []string {
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

var percentLimit string

func init() {

	flag.StringVar(&percentLimit, "L", "100", "limit download %")
	flag.Parse()
}

func main() {

	c := colly.NewCollector()

	data := &Data{}
	networkTest := make([]network, 0)
	locationTests := make(map[Location][]network)
	data.Hosting = make(map[HostingLink]map[Location][]network)

	detail := c.Clone()

	c.OnHTML("tbody tr a.btn.btn-default.btn-block", func(e *colly.HTMLElement) {
		if strings.Contains(e.Attr("href"), "company") {
			detail.Visit(e.Request.AbsoluteURL(e.Attr("href")))
		}
	})

	detail.OnRequest(func(r *colly.Request) {
		log.Println("visiting", r.URL.String())
	})

	detail.OnHTML("body", func(e *colly.HTMLElement) {

		hostingLocations := e.ChildTexts("button.btn.btn-default.btn-block")
		hostingLocations = removeAndModify(hostingLocations, "Full info")
		data.Hosting[HostingLink(e.ChildText("span.pull-right"))] = locationTests

		count := 0
		e.ForEach("div.btn-group.btn-group-justified a", func(_ int, elem *colly.HTMLElement) {
			count++
			n := network{portSpeed: elem.Attr(`title`), linkDownload: elem.Attr(`href`)}
			networkTest = append(networkTest, n)

			for i, hostingLocation := range hostingLocations {
				if hostingLocation != "" {
					locationTests[Location(hostingLocation)] = networkTest
					if count == 3 {
						hostingLocations[i] = ""
						count = 0
						networkTest = []network{}
						continue
					}
					break
				}
			}
		})
	})

	c.Visit("https://looking.house/points.php")

	for _, value := range data.Hosting {
		for location, networkLink := range value {
			spd, _, err := wget.WgetCli([]string{"", "-O", "/dev/null", "-L", percentLimit, networkLink[0].linkDownload})
			if err != nil {
				log.Println()
				log.Println(err)
				continue
			}
			log.Printf("\nLocation: %s, port speed: %s, link: %s, speedtest: %0.2fMbit/s", location, networkLink[0].portSpeed, networkLink[0].linkDownload, spd[0])
		}
	}
}
