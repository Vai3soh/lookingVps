lookingVps

This tool for measure speed test from your connection to VPS. 
Using scraping hosters from site ```https://looking.house/points.php```,
and download file from vps and then save result to *.csv.
The program uses the following libraries(package):

| Package                                   | Change commits                         |
| ----------------------------------------- |----------------------------------------|
| github.com/palvarezcordoba/wget-go        |57a61cabf0721964ea31f017f830d2c240502ad8| 
|											                      |8ee6e3fbe58978228cae9ad434f5a3c2d46502b3|
|											                      |18ab524933b526859b1137887dcdeb6a30184f2d|
|											                      |57a61cabf0721964ea31f017f830d2c240502ad8|
|											                      |8ee6e3fbe58978228cae9ad434f5a3c2d46502b3|
|											                      |19cd35fe6b6562b5f81d28cb83e960b73e151d3b|
|											                      |920b4576c74092bc5b656aff6a5a6e9595d029de|
| github.com/laher/uggo                     |                                        |
| github.com/gocolly/colly/v2               |                                        |


Build:

```git clone github.com/Vai3soh/lookingVps```

make build

run example(download 1000Mb file, and limit 30%):

```./cmd/app/lookingVps -h
Usage of ./cmd/app/lookingVps:
  -D int
    	download file size {10,100,1000 MB} % (default 10)
  -L string
    	limit download % (default "100")
  -S string
    	save to data result in csv file path (default "result.csv")
  -U string
    	user-agent (default "Mozilla/5.0 (Windows NT 6.1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/96.0.4664.45 Safari/537.36")
  -W string
    	save download file, default /dev/null (default "/dev/null")
```

```./cmd/app/lookingVps -D 1000 -L 30```
