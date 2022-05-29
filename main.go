// L_47 first concurrency go program
package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type Download struct {
	Url           string
	TargetPath    string
	TotalSections int
}

func main() {
	startTime := time.Now()
	d := Download{
		// Provide the URL to download,
		Url: "",
		// Provide the target file path with extension, example: sample.mp4
		TargetPath: "",
		// Number of connections to make to the server
		TotalSections: 10,
	}
	err := d.Do()
	if err != nil {
		log.Printf("An error occured, please try again or use another url or extension: %s\n", err)
	}
	fmt.Printf("Download completed in %v seconds\n", time.Now().Sub(startTime).Seconds())
}

func (d Download) Do() error {
	fmt.Println("Checking URL")
	r, err := d.getNewRequest("HEAD")
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	fmt.Printf("Got %v\n", resp.StatusCode)

	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}

	size, err := strconv.Atoi(resp.Header.Get("Content-Length"))
	if err != nil {
		return err
	}
	fmt.Printf("Size is %v bytes\n", size)

	var sections = make([][2]int, d.TotalSections)
	eachSize := size / d.TotalSections
	fmt.Printf("Each size is %v bytes\n", eachSize)


	for i := range sections {
		if i == 0 {
			sections[i][0] = 0
		} else {
			sections[i][0] = sections[i-1][1] + 1
		}

		if i < d.TotalSections-1 {
			sections[i][1] = sections[i][0] + eachSize
		} else {
			sections[i][1] = size - 1
		}
	}

	log.Println(sections)
	var wg sync.WaitGroup
	// download each section concurrently
	for i, s := range sections {
		wg.Add(1)
		go func(i int, s [2]int) {
			defer wg.Done()
			err = d.downloadSection(i, s)
			if err != nil {
				panic(err)
			}
		}(i, s)
	}
	wg.Wait()

	return d.mergeFiles(sections)
}

// Download a single section and save content to a tmp file
func (d Download) downloadSection(i int, c [2]int) error {
	r, err := d.getNewRequest("GET")
	if err != nil {
		return err
	}
	r.Header.Set("Range", fmt.Sprintf("bytes=%v-%v", c[0], c[1]))
	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return err
	}
	if resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("Can't process, response is %v", resp.StatusCode))
	}
	fmt.Printf("Downloaded %v bytes for section %v\n", resp.Header.Get("Content-Length"), i)
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fmt.Sprintf("section-%v.tmp", i), b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (d Download) getNewRequest(method string) (*http.Request, error) {
	r, err := http.NewRequest(
		method,
		d.Url,
		nil,
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "Silly Download Manager v001")
	return r, nil
}

func (d Download) mergeFiles(sections [][2]int) error {
	f, err := os.OpenFile(d.TargetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()
	for i := range sections {
		tmpFileName := fmt.Sprintf("section-%v.tmp", i)
		b, err := ioutil.ReadFile(tmpFileName)
		if err != nil {
			return err
		}
		n, err := f.Write(b)
		if err != nil {
			return err
		}
		err = os.Remove(tmpFileName)
		if err != nil {
			return err
		}
		fmt.Printf("%v bytes merged\n", n)
	}
	return nil
}
