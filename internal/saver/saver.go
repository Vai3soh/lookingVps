package saver

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/Vai3soh/lookingVps/internal/scraping"
)

type Saver struct{}

func NewSaver() *Saver {
	return &Saver{}
}

func (s *Saver) SaveToCSV(data [][]string, csvFilePath string) error {
	file, err := os.Create(csvFilePath)
	if err != nil {
		return fmt.Errorf("cannot create file %s: %w", csvFilePath, err)
	}
	log.Printf("[DEBUG] File %s created successfully.", csvFilePath)
	writer := csv.NewWriter(file)
	err = writer.WriteAll(data)
	if err != nil {
		_ = file.Close()
		return fmt.Errorf("cannot write to file %s: %w", csvFilePath, err)
	}
	writer.Flush()
	err = file.Close()
	if err != nil {
		return fmt.Errorf("cannot close file %s: %w", csvFilePath, err)
	}
	log.Printf("[INFO] CSV saved to %s", csvFilePath)
	return nil
}

func (s *Saver) SortAndSave(data []scraping.SaveData, csvFilePath string) error {
	var writerData [][]string

	writerData = append(writerData, []string{
		"Hosting name", "Country", "Hosting link", "Speedtest link", "Speedtest Mbit/s",
	})

	for _, d := range data {
		if d.SpeedTest != 0 {
			writerData = append(writerData, []string{
				d.HostingName,
				d.Country,
				d.HostingURL,
				d.SpeedTestLink,
				strconv.FormatFloat(d.SpeedTest, 'f', 2, 64),
			})
		}
	}

	if len(writerData) > 1 {
		sort.Slice(writerData[1:], func(i, j int) bool {
			speed1, _ := strconv.ParseFloat(writerData[i+1][4], 64)
			speed2, _ := strconv.ParseFloat(writerData[j+1][4], 64)
			return speed1 > speed2
		})
	}

	return s.SaveToCSV(writerData, csvFilePath)
}
