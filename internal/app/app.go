package app

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Vai3soh/lookingVps/internal/saver"
	"github.com/Vai3soh/lookingVps/internal/scraping"
)

var downloadLimit int    // limit download percentage
var downloadSizeMB int   // 10,100,1000,10000
var csvResultFile string // CSV output file
var logOutputFile string // file path for saving download output
var userAgent string     // user-agent string
var debugMode bool       // debug flag
var contextTimeout int   // dead line timeout in seconds

const mainURL string = "https://looking.house/looking-glass"

func init() {
	flag.IntVar(&downloadLimit, "L", 100, "limit download percentage")
	flag.IntVar(&downloadSizeMB, "D", 100, "download file size in MB. Options: 10, 100, 1000, 10000")
	flag.StringVar(&logOutputFile, "W", "/dev/null", "file path for saving download output")
	flag.StringVar(&csvResultFile, "S", "result.csv", "CSV file path for saving results")
	flag.StringVar(&userAgent, "U", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36", "User-Agent string")
	flag.BoolVar(&debugMode, "debug", false, "enable debug logging")
	flag.IntVar(&contextTimeout, "T", 30, "dead line timeout in seconds")
	flag.Parse()
}

func Run() {
	if debugMode {
		log.Println("[DEBUG] Starting application in debug mode")
	}
	s := saver.NewSaver()
	sc := scraping.NewCollectors(
		[5]string{
			"h1",
			"a[href^='https://looking.house/companies/']",
			"div.location",
			"div.speed-test",
			"a",
		},
		userAgent,
		debugMode,
	)

	cfg := scraping.NewScrapeConfig(
		downloadSizeMB,
		downloadLimit,
		contextTimeout,
		mainURL,
		logOutputFile,
	)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dataCh := make(chan []scraping.SaveData, 1)

	go func() {
		data := sc.ScrapeServerData(ctx, cfg)
		dataCh <- data
	}()

	data := <-dataCh
	log.Println("[INFO] Saving data...")
	if err := s.SortAndSave(data, csvResultFile); err != nil {
		log.Fatalf("Error saving data: %v", err)
	}
	log.Println("[INFO] Data saved successfully")
}
