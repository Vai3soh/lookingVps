package app

import (
	"flag"

	"github.com/Vai3soh/speedtestVps/internal/scraping"
	"github.com/xlab/closer"
)

type HostingLink string

var percentLimit string
var downloadFileSize int
var saveFileCsv string
var wgetSaveFile string
var ua string

const globalUrl string = "https://looking.house/points.php"

func init() {

	flag.StringVar(&percentLimit, "L", "100", "limit download %")
	flag.IntVar(&downloadFileSize, "D", 10, "download file size {10,100,1000 MB} %")
	flag.StringVar(&wgetSaveFile, "W", "/dev/null", "save download file, default /dev/null")
	flag.StringVar(&saveFileCsv, "S", "result.csv", "save to data result in csv file path")
	flag.StringVar(&ua, "U", "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36", "user-agent")

	flag.Parse()
}

func Run() {

	sc := scraping.NewCollectors(
		[5]string{
			"body",
		},
		ua,
	)

	sl := make([]chan []scraping.SaveData, 0)
	sc.GetBufferSize(&sl, globalUrl)

	closer.Bind(func() {
		close(sl[0])
		for element := range sl[0] {
			sorterData(element, saveFileCsv)
		}
	})

	scUrls := scraping.NewCollectors(
		[5]string{
			"body",
			"tbody tr a.btn.btn-default.btn-block",
			"span.pull-right",
			"button.btn.btn-default.btn-block",
			"div.btn-group.btn-group-justified a.btn.btn-default",
		},
		ua,
	)

	data := scUrls.ReadDetailColl(
		downloadFileSize,
		percentLimit,
		globalUrl,
		wgetSaveFile,
		sl[0],
	)

	sorterData(data, saveFileCsv)
	closer.Close()
}
