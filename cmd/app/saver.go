package main

import (
	"encoding/csv"
	"log"
	"os"
	"sort"
	"strconv"

	"github.com/Vai3soh/speedtestVps/internal/scraping"
)

func checkError(message string, err error) {
	if err != nil {
		log.Fatal(message, err)
	}
}

func saveToCsv(data [][]string, filepath string) {
	file, err := os.Create(filepath)
	checkError("cannot create file", err)
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	err = writer.WriteAll(data)
	checkError("cannot write to file", err)
}

func sorterData(data []scraping.SaveData, filepath string) {
	writer := [][]string{}
	for _, e := range data {
		for _, v := range e.DataCenterAndSpeedTestLink {
			for name, ex := range v {
				for _, vl := range ex {
					if vl.SpeedTest != 0 {
						writer = append(writer,
							[]string{
								string(e.HostingLink),
								name,
								vl.LinkDownload,
								strconv.FormatFloat(vl.SpeedTest, 'f', 2, 64),
							},
						)
					}
				}
			}
		}
	}
	sort.Slice(writer, func(i, j int) bool {
		for range writer[i] {
			v1, _ := strconv.ParseFloat(writer[i][3], 64)
			v2, _ := strconv.ParseFloat(writer[j][3], 64)
			return v1 > v2
		}
		return false
	})
	saveToCsv(writer, filepath)
}
