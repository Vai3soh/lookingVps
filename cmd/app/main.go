package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/Vai3soh/speedtestVps/pkg/wget-go"
	"github.com/gocolly/colly/v2"
)

type HostingLink string

var percentLimit string
var downloadFileSize int

type network struct {
	portSpeed    string
	linkDownload string
	speedTest    float64
}

type SaveData struct {
	HostingLink
	DataCenterAndSpeedTestLink []map[string][]network
}

func init() {

	flag.StringVar(&percentLimit, "L", "100", "limit download %")
	flag.IntVar(&downloadFileSize, "D", 10, "download file size {10,100,1000 MB} %")
	flag.Parse()
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

func downloadOrNot(link string) bool {
	switch downloadFileSize {
	case 10:
		if strings.Contains(link, "10.mb") {
			return true
		}
	case 100:
		if strings.Contains(link, "100.mb") {
			return true
		}
	case 1000:
		if strings.Contains(link, "1000.mb") {
			return true
		}
	}
	return false
}

func speedTest(linkDownload string) ([]float64, error) {
	spd, _, err := wget.WgetCli([]string{"", "-O", "/dev/null", "-L", percentLimit, linkDownload})
	if err != nil {
		return nil, err
	}
	return spd, err
}

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func saveToCsv(data [][]string) {
	file, err := os.Create("result.csv")
	checkError("Cannot create file", err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.WriteAll(data)
	checkError("Cannot write to file", err)
}

func main() {

	c := colly.NewCollector()

	s := &SaveData{}
	all := []SaveData{}

	detail := c.Clone()

	c.OnHTML("tbody tr a.btn.btn-default.btn-block", func(e *colly.HTMLElement) {
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

	detail.OnHTML("body", func(e *colly.HTMLElement) {

		hostingLink := e.ChildText("span.pull-right")
		dataCenters := e.ChildTexts("button.btn.btn-default.btn-block")
		dataCenters = removeAndModify(dataCenters, "Full info")
		linkDownloads := e.ChildAttrs("div.btn-group.btn-group-justified a.btn.btn-default", "href")
		titles := e.ChildAttrs("div.btn-group.btn-group-justified a.btn.btn-default", "title")

		merge := [][]network{}
		group := []network{}

		for i := range linkDownloads {

			spd := 0.0
			ok := downloadOrNot(linkDownloads[i])
			if ok {
				spds, err := speedTest(linkDownloads[i])
				if err != nil {
					log.Println()
					log.Println(err)
				} else {
					if len(spds) != 0 {
						spd = spds[0]
					}
				}
			}
			nw := &network{
				portSpeed:    titles[i],
				linkDownload: linkDownloads[i],
				speedTest:    spd,
			}

			group = append(group, *nw)

			if len(group) == 3 {
				merge = append(merge, group)
				group = []network{}
			}
		}

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
	})

	c.Visit("https://looking.house/points.php")
	writer := [][]string{}
	//writer = append(writer, []string{"HostingLink", "DataCentr", "LinkDownload", "SpeedTest Mbit/s"})

	writer = [][]string{}
	for _, e := range all {
		for _, v := range e.DataCenterAndSpeedTestLink {
			for name, ex := range v {
				for _, vl := range ex {
					if vl.speedTest != 0 {
						writer = append(writer,
							[]string{
								string(e.HostingLink),
								name,
								vl.linkDownload,
								strconv.FormatFloat(vl.speedTest, 'f', 2, 64)},
						)
					}
				}
			}
		}
	}

	sort.Slice(writer, func(i, j int) bool {
		for x := 0; x < len(writer[i]); x++ {
			v1, _ := strconv.ParseFloat(writer[i][3], 64)
			v2, _ := strconv.ParseFloat(writer[j][3], 64)
			return v1 > v2
		}
		return false
	})
	saveToCsv(writer)

}
