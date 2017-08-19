package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

const (
	version = "1.0.3"
)

var (
	mainChan chan string
	// Flag for argument input
	configDir = flag.String("config", "./config.json", "dir to config file")
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func filePrefix() string {
	ts := time.Now().Format("20060102-15:04:05.00")
	ts = strings.Replace(ts, ".", ":", 1)
	return ts
}

func writeData(data string) {
	var buffer bytes.Buffer
	s := strings.Split(data, ",")
	fname := monitorMap[s[0]].localFile
	// Build data string
	buffer.WriteString(filePrefix())
	for i := range s {
		if i == 0 {
			continue
		} // Skip device name
		buffer.WriteString(",")
		buffer.WriteString(s[i])
	}
	buffer.WriteString("\n")
	fmt.Printf("Write file %s: %s", fname, buffer.String())
	// Write incoming data to file
	if f, err := os.OpenFile(fname, os.O_APPEND|os.O_WRONLY, 0600); err != nil {
		check(err)
	} else {
		defer f.Close()
		if _, err = f.Write(buffer.Bytes()); err != nil {
			check(err)
		}
	}
}

func main() {
	mainChan = make(chan string)
	fmt.Printf("emc service start, version: %s\n", version)

	fname := "tmp" // FIXME
	go httpHnadler(fname)
	go buildConfig(*configDir)
	go buildMonitor()
	go buildController()

	for {
		writeData(<-mainChan)
	}
}
