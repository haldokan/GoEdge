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

func multiplexTicker(ticker1, ticker2 <-chan Quote) <-chan Quote {
 ch := make(chan Quote)
 go func() {
  for {
   ch <- <-ticker1
  }
 }()
 go func() {
  for {
   ch <- <-ticker2
  }
 }()
 return ch
}

func main() {
 ch := multiplexTicker(ticker("GOOGLE"), ticker("IBM"))
 for i := 0; i < count; i++ {
  fmt.Printf("Quote %v\n", <-ch)
 }
}