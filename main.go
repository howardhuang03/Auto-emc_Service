package main

import (
  "os"
  "io"
  "fmt"
  "time"
  "flag"
  "bytes"
  "strings"
  "io/ioutil"
  "encoding/json"
)

type config struct {
  Device string `json:"device"`
  Id string `json:"id"`
  Key string `json:"key"`
  Interval int `json:interval` // Update per interval * 5min
  Sensors int `json:sensors`
  localFile string
}

type devConf struct {
  Count int // Interval check
}

const (
  version = "1.0.2"
)

var (
  configMaps map[string]config
  devConfMaps map[string]devConf
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
  fname := configMaps[s[0]].localFile
  // Build data string
  buffer.WriteString(filePrefix())
  for i := range s {
    if (i == 0) {continue} // Skip device name
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

func setConfigMaps(file string) map[string]config {
  var c config
  maps := make(map[string]config)

  jsonData, e := ioutil.ReadFile(file)
  if e != nil {
      check(e)
      os.Exit(1)
  }

  jsonParser := json.NewDecoder(bytes.NewReader(jsonData))
  for {
    if err := jsonParser.Decode(&c); err == io.EOF {
      break
    } else if err != nil {
      check(e)
    }

    if err := os.MkdirAll(c.Device, 0777); err != nil {
      fmt.Println("Mkdir %s failed: %v", c.Device, err)
    }

    fname := fmt.Sprintf("%s/%s.csv", c.Device, time.Now().Format("20060102"))
    file, err := os.Create(fname)
    if err != nil {
      fmt.Println("create %s fail, err: %v", fname, err);
    }

    c.localFile = fname
    defer file.Close()

    maps[c.Device] = c
  }

  fmt.Println("maps:", maps)
  return maps
}

func setDevConfMaps(m map[string]config) map[string]devConf {
  var c devConf
  maps := make(map[string]devConf)

  for k, v := range m {
    c.Count = v.Interval - 1 // Update first data
    maps[k] = c
  }

  return maps
}

func main() {
  mainChan = make(chan string)
  configMaps = setConfigMaps(*configDir)
  devConfMaps = setDevConfMaps(configMaps)

  fmt.Printf("emc service start, version: %s\n", version)

  fname := "tmp" // FIXME
  go httpHnadler(fname)
  go mqttService()

  for {
    writeData(<- mainChan)
  }
}
