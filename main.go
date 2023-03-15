package main

import (
	"bufio"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/briandowns/spinner"
)

// Checks HTTP status of all paths
func GetPaths(host string, filename string, pathList []string, dirList []string, client *http.Client) ([]string, []string) {
	target := host + filename
	resp, err := client.Get(target)
	if err != nil {
		log.Fatalln(err)
	}

	resp.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:75.0) Gecko/20100101 Firefox/75.0")

	var re = regexp.MustCompile(`.css|.png|.jpg|.gif|.jpeg|.ttf|.hlp|.gid|.db`)

	if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Fatalln(err)
		}

		sb := string(body)

		scanner := bufio.NewScanner(strings.NewReader(sb))

		for scanner.Scan() {
			sp := strings.Split(scanner.Text(), "/")
			// If it's not a directory or a file, just move on
			if strings.Split(scanner.Text(), "")[0] == ":" {
				continue
				// If it's a directory, move on for now
			} else if strings.Contains(scanner.Text(), "D/") {
				dirList = append(dirList, "/"+sp[1])
				//If it's a file, append it to the slice unless it containes an ignored extension
			} else if len(sp) > 1 {
				sl := strings.ToLower(scanner.Text())
				if !re.MatchString(sl) {
					pathList = append(pathList, ("/" + sp[1]))
				}
			}
		}
	}
	return pathList, dirList
}

// Takes a slice of paths and returns a slice of paths that return 200 codes
func GetValidPaths(host string, pathList []string, threads int, client *http.Client) []string {
	var validPaths []string

	spin := spinner.New(spinner.CharSets[1], 100*time.Millisecond)
	spin.Prefix = "[*] Testing for valid file paths "
	spin.Start()

	sem := make(chan bool, threads)
	mut := &sync.Mutex{}

	for _, path := range pathList {
		sem <- true
		go func(path string) {
			resp, _ := client.Get(host + path)
			if resp.StatusCode >= 200 && resp.StatusCode <= 299 {
				mut.Lock()
				validPaths = append(validPaths, path)
				mut.Unlock()
			}
			<-sem
		}(path)
	}

	for i := 0; i < cap(sem); i++ {
		sem <- true
	}

	spin.Stop()

	return validPaths
}

// Download files from slice of succesful requests
func DownloadFiles(host string, pathList []string, client *http.Client) {
	u, _ := url.Parse(host)

	// Create base directory for target
	os.Mkdir(u.Host, os.ModePerm)

	for _, fullpath := range pathList {
		f, _ := url.Parse(fullpath)

		fileName := path.Base(fullpath)
		filePath := path.Dir(fullpath)

		// Take the full path of the file and mkdir for each leg of the path
		os.MkdirAll(u.Host+filePath, os.ModePerm)

		fullFilePath := u.Scheme + "://" + u.Host + f.Path

		// Create blank file
		file, err := os.Create(u.Host + filePath + "/" + fileName)
		if err != nil {
			log.Fatal(err)
		}
		// Put content on file
		resp, err := client.Get(fullFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		size, err := io.Copy(file, resp.Body)

		defer file.Close()

		fmt.Printf("Downloaded file %s with size %d\n", fileName, size)
	}
}

func CreateClient(proxy string) http.Client {
	if proxy == "NOPROXY" {
		tr := &http.Transport{
			MaxIdleConns:        30,
			MaxIdleConnsPerHost: 30,
			IdleConnTimeout:     30 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		}

		client := &http.Client{Transport: tr}

		return *client

	} else {
		urlProxy, _ := url.Parse(proxy)

		tr := &http.Transport{
			MaxIdleConns:        30,
			MaxIdleConnsPerHost: 30,
			IdleConnTimeout:     30 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			Proxy:               http.ProxyURL(urlProxy),
		}

		client := &http.Client{Transport: tr}

		return *client
	}
}

func main() {
	hostPtr := flag.String("host", "NOHOST", "Url to target. Example: https://example.com")
	proxyPtr := flag.String("proxy", "NOPROXY", "Proxy host and port. Example: http://127.0.0.1:8080")
	threadsPtr := flag.Int("threads", 20, "Number of concurrent threads to run. Example: 100")
	hostListPtr := flag.String("list", "NOLIST", "List of hosts. Example: host_list.txt")
	flag.Parse()

	var hosts []string

	if *hostPtr == "NOHOST" && *hostListPtr == "NOLIST" {
		fmt.Println("A host or host list is required.")
		os.Exit(0)
	} else if *hostListPtr != "NOLIST" {
		file, err := os.Open(*hostListPtr)
		if err != nil {
			fmt.Println("Error opening file.")
			fmt.Println(err)
			os.Exit(0)
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			hosts = append(hosts, scanner.Text())
		}
	} else {
		hosts = append(hosts, *hostPtr)
	}

	client := CreateClient(*proxyPtr)

	filenames := []string{
		"/CVS/Entries",
		"/CVS/Base/",
		"/CVS/Baserev",
		"/CVS/Baserev.tmp",
		"/CVS/Checkin.prog",
		"/CVS/Entries.Backup",
		"/CVS/Entries.Log",
		"/CVS/Entries.Static",
		"/CVS/Notify",
		"/CVS/Notify.tmp",
		"/CVS/Repository",
		"/CVS/Root",
		"/CVS/Tag",
		"/CVS/Template",
		"/CVS/Update.prog",
	}

	for _, host := range hosts {
		fmt.Printf("[+] Testing %s\n", host)
		var pathList []string
		var dirList []string

		for _, filename := range filenames {
			pathList, dirList = GetPaths(host, filename, pathList, dirList, &client)
		}
		fmt.Printf("\n[!] Found %d filepaths. Attempting to download valid paths.\n", len(pathList))
		validPaths := GetValidPaths(host, pathList, *threadsPtr, &client)
		DownloadFiles(host, validPaths, &client)
		if len(dirList) > 0 {
			fmt.Printf("\n[!] Directories found: \n")
			for _, dir := range dirList {
				fmt.Println(dir)
			}
		} else {
			fmt.Println("\n[-] No directories found.")
		}
		fmt.Println("===============================")

	}
	fmt.Println("Done")
}
