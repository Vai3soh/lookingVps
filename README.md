lookingVps

This tool for measure speed test from your connection to VPS. 
Using scraping hosters from site ```https://looking.house/looking-glass```,
and download file from vps and then save result to *.csv.
The program uses the following libraries(package):

| Package                                   | Change commits                         |
| ----------------------------------------- |----------------------------------------|
| github.com/gocolly/colly/v2               |                                        |


Build:

```git clone github.com/Vai3soh/lookingVps```

make build

run example(download 1000Mb file, and limit 30%):

```./cmd/app/lookingVps -h
Usage of ./cmd/app/lookingVps:
  -D int
    	download file size in MB. Options: 10, 100, 1000, 10000 (default 100)
  -L int
    	limit download percentage (default 100)
  -S string
    	CSV file path for saving results (default "result.csv")
  -T int
    	dead line timeout in seconds (default 30)
  -U string
    	User-Agent string (default "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
  -W string
    	file path for saving download output (default "/dev/null")
  -debug
    	enable debug logging
```

```./cmd/app/lookingVps -D 1000 -L 30```
