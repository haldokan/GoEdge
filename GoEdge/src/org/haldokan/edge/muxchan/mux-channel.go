package main

import (
 "fmt"
 "math/rand"
 "time"
)

type Quote struct {
 stock string
 px    float64
}

const count = 10

func ticker(stock string) <-chan Quote {
 ch := make(chan Quote)
 go func() {
  for {
   ch <- Quote{stock, rand.Float64()}
   time.Sleep(time.Duration(rand.Intn(100)) * time.Millisecond)
  }
 }()
 return ch
}

func multiplexTicker(ticker1, ticker2 <-chan Quote, signal chan bool) <-chan Quote {
 ch := make(chan Quote)
 go func() {
  for {
   select {
   case t1 := <-ticker1:
    ch <- t1
   case t2 := <-ticker2:
    ch <- t2
   case <-signal:
    fmt.Println("Lassen wir vor here weggehen!!!")
    signal <- true
    return
   }
  }
 }()
 return ch
}

func main() {
 signal := make(chan bool)
 ch := multiplexTicker(ticker("GOOGLE"), ticker("IBM"), signal)
 for i := 0; i < count; i++ {
  fmt.Printf("Quote %v\n", <-ch)
 }
 signal <- true
 fmt.Println("finished ", <- signal)
}