// Wgetter downloads and saves/pipes HTTP requests
package wget

import (
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Vai3soh/speedtestVps/pkg/uggo"
)

//TODO
// ftp (3rd party supp?)
// clobber behaviour
// limit-rate
// timeouts - connect,read,dns?
// wait,waitretry
// proxies/?
// quota/?
// user/password/ask-password
// certificates/no-check-certificate ...
// exit statuses
// recursive downloads
// timestamping
// wgetrc
type Wgetter struct {
	IsContinue bool
	// should be set explicitly to false when running from CLI. uggo will detect as best as possible
	AlwaysPipeStdin      bool
	OutputFilename       string
	Timeout              int //TODO
	PercentLimit         int64
	Retries              int  //TODO
	IsVerbose            bool //todo
	DefaultPage          string
	UserAgent            string //todo
	ProxyUser            string //todo
	ProxyPassword        string //todo
	Referer              string //todo
	SaveHeaders          bool   //todo
	PostData             string //todo
	HttpUser             string //todo
	HttpPassword         string //todo
	IsNoCheckCertificate bool
	SecureProtocol       string

	links []string
}

const (
	VERSION              = "0.5.0"
	FILEMODE os.FileMode = 0660
)

//Factory for wgetter which outputs to Stdout
func WgetToOut(urls ...string) *Wgetter {
	wgetter := Wget(urls...)
	wgetter.OutputFilename = "-"
	return wgetter
}

// Factory for wgetter
func Wget(urls ...string) *Wgetter {
	wgetter := new(Wgetter)
	wgetter.links = urls
	if len(urls) == 0 {
		wgetter.AlwaysPipeStdin = true
	}
	return wgetter
}

// CLI invocation for wgetter
func WgetCli(call []string) ([]float64, int, error) {
	inPipe := os.Stdin
	outPipe := os.Stdout
	errPipe := os.Stderr
	wgetter := new(Wgetter)
	wgetter.AlwaysPipeStdin = false
	code, err := wgetter.ParseFlags(call, errPipe)
	if err != nil {
		return []float64{0.0}, code, err
	}
	spd, code, err := wgetter.Exec(wgetter.PercentLimit, inPipe, outPipe, errPipe)
	return spd, code, err
}

// Name() returns the name of the util
func (tail *Wgetter) Name() string {
	return "wget"
}

// Parse CLI flags
func (w *Wgetter) ParseFlags(call []string, errPipe io.Writer) (int, error) {

	flagSet := uggo.NewFlagSetDefault("wget", "[options] URL", VERSION)
	flagSet.SetOutput(errPipe)
	flagSet.AliasedBoolVar(&w.IsContinue, []string{"c", "continue"}, false, "continue")
	flagSet.AliasedStringVar(&w.OutputFilename, []string{"O", "output-document"}, "", "specify filename")
	flagSet.AliasedInt64Var(&w.PercentLimit, []string{"L", "percent-limit"}, 100, "stop download file - percent limit")
	flagSet.StringVar(&w.DefaultPage, "default-page", "index.html", "default page name")
	flagSet.BoolVar(&w.IsNoCheckCertificate, "no-check-certificate", false, "skip certificate checks")

	//some features are available in go-1.2+ only
	extraOptions(flagSet, w)
	err, code := flagSet.ParsePlus(call[1:])
	if err != nil {
		return code, err
	}

	//fmt.Fprintf(errPipe, "%+v\n", w)
	args := flagSet.Args()
	if len(args) < 1 {
		flagSet.Usage()
		return 1, errors.New("not enough args")
	}
	if len(args) > 0 {
		w.links = args
		//return wget(links, w)
		return 0, nil
	} else {
		if w.AlwaysPipeStdin || uggo.IsPipingStdin() {
			//check STDIN
			//return wget([]string{}, options)
			return 0, nil
		} else {
			//NOT piping.
			flagSet.Usage()
			return 1, errors.New("not enough args")
		}
	}
}

// Perform the wget ...
func (w *Wgetter) Exec(percentLimit int64, inPipe io.Reader, outPipe io.Writer, errPipe io.Writer) ([]float64, int, error) {
	spd := []float64{}
	for _, link := range w.links {
		spd_, err := wgetOne(link, percentLimit, w, outPipe, errPipe)
		if err != nil {
			fmt.Printf("download link %s error: %s\n", link, err)
		}

		spd = append(spd, *spd_)
	}
	return spd, 0, nil
}

func tidyFilename(filename, defaultFilename string) string {
	//invalid filenames ...
	if filename == "" || filename == "/" || filename == "\\" || filename == "." {
		filename = defaultFilename
		//filename = "index"
	}
	return filename
}

func wgetOne(link string, percentLimit int64, options *Wgetter, outPipe io.Writer, errPipe io.Writer) (*float64, error) {
	if !strings.Contains(link, ":") {
		link = "http://" + link
	}
	startTime := time.Now()
	request, err := http.NewRequest("GET", link, nil)
	//resp, err := http.Get(link)
	if err != nil {
		return nil, err
	}

	filename := ""
	//include stdout
	if options.OutputFilename != "" {
		filename = options.OutputFilename
	}

	tr, err := getHttpTransport(options)
	if err != nil {
		return nil, err
	}
	client := &http.Client{Transport: tr}

	//continue from where we left off ...
	if options.IsContinue {
		if options.OutputFilename == "-" {
			return nil, errors.New("continue not supported while piping")
		}
		//not specified
		if filename == "" {
			filename = filepath.Base(request.URL.Path)
			filename = tidyFilename(filename, options.DefaultPage)
		}
		if !strings.Contains(filename, ".") {
			filename = filename + ".html"
		}
		fi, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		from := fi.Size()
		rangeHeader := fmt.Sprintf("bytes=%d-", from)
		request.Header.Add("Range", rangeHeader)
	}
	if options.IsVerbose {
		for headerName, headerValue := range request.Header {
			fmt.Fprintf(errPipe, "Request header %s: %s\n", headerName, headerValue)
		}
	}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	fmt.Fprintf(errPipe, "Http response status: %s\n", resp.Status)
	if options.IsVerbose {
		for headerName, headerValue := range resp.Header {
			fmt.Fprintf(errPipe, "Response header %s: %s\n", headerName, headerValue)
		}
	}
	lenS := resp.Header.Get("Content-Length")
	length := int64(-1)
	if lenS != "" {
		length, err = strconv.ParseInt(lenS, 10, 32)
		if err != nil {
			return nil, err
		}
	}

	typ := resp.Header.Get("Content-Type")
	fmt.Fprintf(errPipe, "Content-Length: %v Content-Type: %s\n", lenS, typ)

	if filename == "" {
		filename, err = getFilename(request, resp, options, errPipe)
		if err != nil {
			return nil, err
		}
	}

	contentRange := resp.Header.Get("Content-Range")
	rangeEffective := false
	if contentRange != "" {
		//TODO parse it?
		rangeEffective = true
	} else if options.IsContinue {
		fmt.Fprintf(errPipe, "Range request did not produce a Content-Range response\n")
	}
	var out io.Writer
	var outFile *os.File
	if filename != "-" {
		fmt.Fprintf(errPipe, "Saving to: '%v'\n\n", filename)
		var openFlags int
		if options.IsContinue && rangeEffective {
			openFlags = os.O_WRONLY | os.O_APPEND
		} else {
			openFlags = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		}
		outFile, err = os.OpenFile(filename, openFlags, FILEMODE)
		if err != nil {
			return nil, err
		}
		defer outFile.Close()
		out = outFile
	} else {
		//save to outPipe
		out = outPipe
	}
	buf := make([]byte, 4068)
	tot := int64(0)
	i := 0

	for {
		// read a chunk
		n, err := resp.Body.Read(buf)
		if err != nil && err != io.EOF {
			return nil, err
		}
		if n == 0 {
			break
		}
		tot += int64(n)

		// write a chunk
		if _, err := out.Write(buf[:n]); err != nil {
			return nil, err
		}
		i += 1
		if length > -1 {
			if length < 1 {
				fmt.Fprintf(errPipe, "\r     [ <=>                                  ] %d\t-.--KB/s eta ?s             ", tot)
			} else {
				//show percentage
				perc := (100 * tot) / length
				prog := progress(perc)
				nowTime := time.Now()
				totTime := nowTime.Sub(startTime)
				spd := (float64(tot/1000) / totTime.Seconds()) * 0.008
				remKb := float64(length-tot) / float64(1000)
				eta := remKb / spd
				fmt.Fprintf(errPipe, "\r%3d%% [%s]          \t%0.2fMbit/s eta %0.1fs             ", perc, prog, spd, eta)
			}
		} else {
			//show dots
			if math.Mod(float64(i), 20) == 0 {
				fmt.Fprint(errPipe, ".")
			}
		}
	}
	nowTime := time.Now()
	totTime := nowTime.Sub(startTime)
	spd := (float64(tot/1000) / totTime.Seconds()) * 0.008
	if length < 1 {
		fmt.Fprintf(errPipe, "\r     [ <=>                                  ] %d\t-.--KB/s in %0.1fs             ", tot, totTime.Seconds())
		fmt.Fprintf(errPipe, "\n (%0.2fMbit/s) - '%v' saved [%v]\n", spd, filename, tot)
	} else {
		perc := (100 * tot) / length
		prog := progress(perc)
		fmt.Fprintf(errPipe, "\r%3d%% [%s] %d\t%0.2fMbit/s in %0.1fs             ", perc, prog, tot, spd, totTime.Seconds())
		fmt.Fprintf(errPipe, "\n '%v' saved [%v/%v]\n", filename, tot, length)
	}
	if err != nil {
		return nil, err
	}
	if outFile != nil {
		err = outFile.Close()
	}
	return &spd, err
}

func progress(perc int64) string {
	equalses := perc * 38 / 100
	if equalses < 0 {
		equalses = 0
	}
	spaces := 38 - equalses
	if spaces < 0 {
		spaces = 0
	}
	prog := strings.Repeat("=", int(equalses)) + ">" + strings.Repeat(" ", int(spaces))
	return prog
}

func getFilename(request *http.Request, resp *http.Response, options *Wgetter, errPipe io.Writer) (string, error) {
	filename := filepath.Base(request.URL.Path)

	if !strings.Contains(filename, ".") {
		//original link didnt represent the file type. Try using the response url (after redirects)
		filename = filepath.Base(resp.Request.URL.Path)
	}
	filename = tidyFilename(filename, options.DefaultPage)

	if !strings.Contains(filename, ".") {
		ct := resp.Header.Get("Content-Type")
		//println(ct)
		ext := "htm"
		mediatype, _, err := mime.ParseMediaType(ct)
		if err != nil {
			fmt.Fprintf(errPipe, "mime error: %v\n", err)
		} else {
			fmt.Fprintf(errPipe, "mime type: %v (from Content-Type %v)\n", mediatype, ct)
			slash := strings.Index(mediatype, "/")
			if slash != -1 {
				_, sub := mediatype[:slash], mediatype[slash+1:]
				if sub != "" {
					ext = sub
				}
			}
		}
		filename = filename + "." + ext
	}
	_, err := os.Stat(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return filename, nil
		} else {
			return "", err
		}
	} else {
		num := 1
		//just stop after 100
		for num < 100 {
			filenameNew := filename + "." + strconv.Itoa(num)
			_, err := os.Stat(filenameNew)
			if err != nil {
				if os.IsNotExist(err) {
					return filenameNew, nil
				} else {
					return "", err
				}
			}
			num += 1
		}
		return filename, errors.New("stopping after trying 100 filename variants")
	}
}
