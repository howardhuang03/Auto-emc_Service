package main

import (
  "os"
  "fmt"
)

var localFile *os.File

func main() {
  fname := fmt.Sprintf("%s.csv", filePrefix())
  var err error
  localFile, err = os.Create(fname)
  check(err)

  go httpHnadler(fname)
  mqttService()
}
