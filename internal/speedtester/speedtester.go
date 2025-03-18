package speedtester

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type SpeedTester struct {
	client *http.Client
	debug  bool
}

func NewSpeedTester(timeoutSec int, debug bool) *SpeedTester {
	return &SpeedTester{
		client: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
		debug: debug,
	}
}

func (st *SpeedTester) getContentLength(ctx context.Context, link string) (int64, error) {
	if st.debug {
		log.Printf("[DEBUG] Performing HEAD request for: %s", link)
	}
	req, err := http.NewRequestWithContext(ctx, "HEAD", link, nil)
	if err != nil {
		return 0, err
	}
	resp, err := st.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	cl := resp.ContentLength
	if cl <= 0 {
		return 0, errors.New("could not determine content length from HEAD request")
	}
	return cl, nil
}

func (st *SpeedTester) setupOutput(wgetOutputFile string) (io.Writer, func(), error) {
	if wgetOutputFile == "/dev/null" {
		return io.Discard, func() {}, nil
	}
	file, err := os.Create(wgetOutputFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create output file: %w", err)
	}
	cleanup := func() {
		file.Close()
	}
	return file, cleanup, nil
}

func (st *SpeedTester) reportProgress(ctx context.Context, start time.Time, contentLength int64, totalRead *int64, stopCh chan struct{}) {
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	barWidth := 50
	for {
		select {
		case <-ticker.C:
			progress := float64(*totalRead) / float64(contentLength)
			percent := progress * 100
			filled := int(progress * float64(barWidth))
			bar := strings.Repeat("=", filled) + strings.Repeat(" ", barWidth-filled)
			elapsed := time.Since(start).Seconds()
			if elapsed <= 0 {
				elapsed = 1
			}
			speed := (float64(*totalRead) / elapsed * 8) / 1e6

			fmt.Printf("\rProgress: [%s] %.2f%%, Speed: %.2f Mbit/s", bar, percent, speed)
		case <-ctx.Done():
			fmt.Println()
			return
		case <-stopCh:
			fmt.Println()
			return
		}
	}
}

func (st *SpeedTester) downloadData(ctx context.Context, link string, targetBytes int64, wgetOutputFile string, contentLength int64) (int64, float64, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", link, nil)
	if err != nil {
		return 0, 0, err
	}

	start := time.Now()
	resp, err := st.client.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()

	outputWriter, cleanup, err := st.setupOutput(wgetOutputFile)
	if err != nil {
		return 0, 0, err
	}
	defer cleanup()

	buf := make([]byte, 32*1024)
	var totalRead int64
	stopCh := make(chan struct{})

	go st.reportProgress(ctx, start, contentLength, &totalRead, stopCh)

	for {
		select {
		case <-ctx.Done():
			close(stopCh)
			return totalRead, time.Since(start).Seconds(), ctx.Err()
		default:
			if totalRead >= targetBytes {
				close(stopCh)
				return totalRead, time.Since(start).Seconds(), nil
			}
			n, err := resp.Body.Read(buf)
			if n > 0 {
				totalRead += int64(n)
				if _, wErr := outputWriter.Write(buf[:n]); wErr != nil {
					close(stopCh)
					return totalRead, time.Since(start).Seconds(), fmt.Errorf("error writing to output: %w", wErr)
				}
			}
			if err == io.EOF {
				close(stopCh)
				return totalRead, time.Since(start).Seconds(), nil
			}
			if err != nil {
				close(stopCh)
				return totalRead, time.Since(start).Seconds(), err
			}
		}
	}
}

func (st *SpeedTester) calculateSpeed(totalRead int64, elapsed float64) float64 {
	return (float64(totalRead) / elapsed * 8) / 1e6 // Speed in Mbit/s
}

func (st *SpeedTester) PerformSpeedTest(ctx context.Context, link string, downloadLimit int, wgetOutputFile string) (float64, error) {
	if st.debug {
		log.Printf("[DEBUG] Starting speed test for: %s with download limit: %d%%", link, downloadLimit)
	}
	contentLength, err := st.getContentLength(ctx, link)
	if err != nil {
		fmt.Println()
		return 0.0, err
	}

	targetBytes := int64(float64(contentLength) * (float64(downloadLimit) / 100.0))
	if targetBytes <= 0 {
		return 0.0, errors.New("computed target download size is zero")
	}
	if st.debug {
		log.Printf("[DEBUG] Content-Length: %d bytes, target: %d bytes (%d%%)", contentLength, targetBytes, downloadLimit)
	}

	totalRead, elapsed, err := st.downloadData(ctx, link, targetBytes, wgetOutputFile, contentLength)
	if err != nil {
		fmt.Println()
		return 0.0, err
	}
	if elapsed <= 0 {
		fmt.Println()
		return 0.0, errors.New("download duration is zero, cannot calculate speed")
	}
	finalSpeed := st.calculateSpeed(totalRead, elapsed)
	if st.debug {
		log.Printf("[DEBUG] Speed test result for %s: %.2f Mbit/s", link, finalSpeed)
	}
	return finalSpeed, nil
}
