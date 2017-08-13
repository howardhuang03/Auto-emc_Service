package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

var (
	monitorMap    map[string]monitor
	controllerMap map[string]controller
)

type monitor struct {
	Device    string `json:"device"`
	Id        string `json:"id"`
	Key       string `json:"key"`
	Interval  int    `json:"interval"` // Update per interval * 5min
	Sensors   int    `json:"sensors"`
	localFile string
}

type timer struct {
	Time     string `json:"time"`
	Interval int    `json:"interval"`
}

type controller struct {
	Device string  `json:"device"`
	Timer  []timer `json:"timer"`
}

type config struct {
	Monitors    []monitor    `json:"monitor"`
	Controllers []controller `json:"controller"`
}

func buildMonitorMap(Config config) map[string]monitor {
	mmap := make(map[string]monitor)
	for i := 0; i < len(Config.Monitors); i++ {
		m := Config.Monitors[i]
		if err := os.MkdirAll(m.Device, 0777); err != nil {
			fmt.Println("Mkdir %s failed: %v", m.Device, err)
		}

		fname := fmt.Sprintf("%s/%s.csv", m.Device, time.Now().Format("20060102"))
		file, err := os.Create(fname)
		if err != nil {
			fmt.Println("create %s fail, err: %v", fname, err)
		}

		m.localFile = fname
		defer file.Close()

		mmap[m.Device] = m
	}

	fmt.Println("monitorMap:", mmap)
	return mmap
}

func buildControllerMap(Config config) map[string]controller {
	cmap := make(map[string]controller)
	for i := 0; i < len(Config.Controllers); i++ {
		c := Config.Controllers[i]
		cmap[c.Device] = c
	}

	fmt.Println("controllerMap:", cmap)
	return cmap
}

func buildConfig(file string) {
	var c config

	jsonData, e := ioutil.ReadFile(file)
	if e != nil {
		check(e)
		os.Exit(1)
	}

	json.Unmarshal(jsonData, &c)

	monitorMap = buildMonitorMap(c)
	controllerMap = buildControllerMap(c)
}
