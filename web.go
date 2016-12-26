package main

import (
  "fmt"
  "net/http"
  "io/ioutil"

  ROUTER "github.com/julienschmidt/httprouter"
)

func httpHnadler (fname string) {
  // Instantiate a new router
  r := ROUTER.New()

  // Add a handler on /test
  r.GET("/test", func(w http.ResponseWriter, r *http.Request, _ ROUTER.Params) {
    // Read sensor data from file
    dat, err := ioutil.ReadFile(fname)
    check(err)
    fmt.Fprint(w, string(dat))
  })

  // Fire up the server
  http.ListenAndServe("localhost:3000", r)
}
