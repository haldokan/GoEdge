import (
 "fmt"
 "haldokan/org/edge/utils"
 "log"
 "net/http"
 "time"
 "os"
)

type String string

type Struct struct {
 Stock string
 Px    float64
 When  time.Time
}

type Counter struct {
 count int
}

type Counter1 int

func (stri String) ServeHTTP(w http.ResponseWriter, r *http.Request) {
 fmt.Fprint(w, stri)
}

func (stru *Struct) ServeHTTP(w http.ResponseWriter, r *http.Request) {
 fmt.Fprint(w, *stru)
}

func (c *Counter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
 c.count++
 fmt.Fprintf(w, "counter = %d\n", c.count) 
}

func (c *Counter1) ServeHTTP(w http.ResponseWriter, r *http.Request) {
 *c++
 fmt.Fprintf(w, "counter = %d\n", *c) 
 
}

func OsServer(w http.ResponseWriter, r *http.Request) {
 fmt.Fprintf(w, "prog invoked with args %v", os.Args)
}

func main() {
 fmt.Println(utils.Reverse("!oG, olleH"))
 // your http.Handle calls here
 http.Handle("/string", String("Well hello there!!!"))
 http.Handle("/struct", &Struct{"IBM", 123.645, time.Now()})
 http.Handle("/counter", &Counter{})
 c1 := Counter1(1)
 http.Handle("/counter1", &c1)
 // HandlerFunc is a type func that is initialized with a func
 http.HandleFunc("/osargs", OsServer)
 // or we can call handle with the args and func
// http.Handle("/osargs", http.HandlerFunc(OsServer))

 log.Fatal(http.ListenAndServe("localhost:4000", nil))
}

package utils

func Reverse(s string) string {
 r := []rune(s)
 for i, j := 0, len(r)-1; i < len(r)/2; i, j = i+1, j-1 {
  r[i], r[j] = r[j], r[i]
 }
 return string(r)
}

package utils

import (
 "testing"
)

func TestReverse(t *testing.T) {
 cases := []struct {
  in, want string
 }{
  {"Hello, world", "dlrow ,olleH"},
  {"Hello, world ,olleH"},
  {"", ""},
 }
 for _, c := range cases {
  got := Reverse(c.in)
  if got != c.want {
   t.Errorf("Reverse(%q) == %q, want %q", c.in, got, c.want)
  }
 }
}