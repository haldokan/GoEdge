package main

import (
 "errors"
 "fmt"
 "math/rand"
 "strconv"
 "time"
)
/*
This RSS server, based on a talk given by a Google engineer, shows advanced usage of concurrency in GO using "chan". It also
demonstrate how structs implement interfaces.

The RSS server fetches RSS updates from multiple RSS channels (dummy ones) concurrently and merges the updates into one update "chan".
Each RSS channel fetcher implements a staggering policy for fetching updates where by it maintains a fixed size queue of pending updates.
Fetching happens only when the queue size is less than the max size and the next fetching time is due.

My next goal is to implement this server in Java in order to appreciate what having "chan" as part of the language contributes to 
making hard-to-implement constructs (in Java) relatively easy to do in GO even for a beginner.
*/

// can also simply use rand.Seed(int64)
var r *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

// RSS update has id, title and the name of the RSS channel (not the GO "chan"; naming is unfortunate tho)
type Item struct{ ID, Title, Chan string }

// RSS fetcher returns a list of RSS updates, the time the next fetch is due, or an error if fetching fails.
// This struct will be implemented by struct DataFetched further down
type Fetcher interface {
 fetch() (items []Item, next time.Time, err error)
}

// RSS fetched batch has a list of items, the time the next fetch is due, and any errors.
// This struct will implement the Fetcher interface further down
type DataFetched struct {
 items []Item
 next  time.Time
 err   error
}

// RSS channel (feed would be a better name) subscriber interface has a func that returns a "chan" of update items,
// a func to close the subscriber, and a hard-to-understand chan of chan of error: basically we are passing a chan to a chan
// I will try to explain that later. This interface will be implemented by by struct Subscription further down
type Subscriber interface {
 UpdatesStream() <-chan Item
 Close() error
 ClosingChan() chan chan error
}

// Subscription has a fetcher, an update chan of items and the curious chan of char of erro
type Subscription struct {
 fetcher Fetcher
 updates chan Item
 closeit chan chan error
}

// The struct Subscription implementation of the Subscriber interface: the next 3 funcs 
func (s *Subscription) UpdatesStream() <-chan Item {
 return s.updates
}

func (s *Subscription) ClosingChan() chan chan error {
 return s.closeit
}

func (s *Subscription) Close() error {
 ch := make(chan error)
 s.closeit <- ch

 return <-ch
}

// func Subscribe takes a fetcher and returns a Subscription (which implements Subscriber as we saw earlier)
func Subscribe(fetcher Fetcher) Subscriber {
 // here we create a pointer to Subscription by assigning the passed fetcher and making an update and an error chan
 s := &Subscription{
  fetcher: fetcher, updates: make(chan Item), closeit: make(chan chan error),
 }
 // start a go routine to serve data using the Subscription fetcher and populating the the fetched data in the struct
 // now how we simply indicate that a func "serve" has a target that is struct Subscription (have a look at the "serve" func)
 go s.serve()
 return s
}

// The merger function takes a list of SubscriptionS and merge their updates into one outgoing updates chan. It also accepts
// close whereupon it closes all the merged subscriptions and itself finally. Here the chan of chan we talked about comes to play.
// Look at "case echan" (error chan). What this case receives is actually a chan. This enables each subscriber to send its close
// request on its own chan (TODO: provide further details)
// Going to assume 3 subscriptions to avoid using reflection (func Select(cases []SelectCase) (chosen int, recv Value, recvOK bool))
func Merge(subs ...Subscriber) Subscriber {
 s := &Subscription{updates: make(chan Item), closeit: make(chan chan error)}
 go func() {
  for {
   select {
   case d1 := <-subs[0].UpdatesStream():
    s.updates <- d1
   case d2 := <-subs[1].UpdatesStream():
    s.updates <- d2
   case d3 := <-subs[2].UpdatesStream():
    s.updates <- d3
   case echan := <-s.closeit:
    err := subs[0].Close()
    fmt.Printf("closed %v\n", err)
    err = subs[1].Close()
    fmt.Println("closed", err)
    err = subs[2].Close()
    fmt.Println("closed", err)
    echan <- errors.New("Closed merger")
    close(s.updates)
    return
   }
  }
 }()
 return s
}

// struct DataFetched implements the Fetcher interface. It unpacks the fetched data and returns it
func (d *DataFetched) fetch() (items []Item, next time.Time, err error) {
 d.next = time.Now().Add(time.Duration(2 * time.Second))
 return d.items, d.next, d.err
}

// A Generator function that returns dummy FetcherS. It takes an RSS channel (or feed) and randomly generate updates or errors.
// As you might expect from your RSS feed it returns Buddhist wisdom in French!
func Fetch(channel string) Fetcher {
 var (
  items []Item
  err   error
 )
 if rand.Intn(3) == 0 {
  err = errors.New(fmt.Sprintf("channel '%s' - error %s", channel, "Die Beide Gotte einen mit einander Schlag haben"))
 } else {
  items = []Item{{"item" + strconv.Itoa(rand.Intn(50)), "En premier lieu, nous concevons le <<moi>>, et nous y attachons.", channel},
   {"item" + strconv.Itoa(rand.Intn(50)), "Puis nous concevons le <<mien>>, et nous attachons au monde materiel.", channel},
   {"item" + strconv.Itoa(rand.Intn(50)), "Comme l'eau captive de la roue du moulin, nous tournons en rond, impuissants.", channel},
   {"item" + strconv.Itoa(rand.Intn(50)), "Je rend hommage a la compassion qui embrasse tous les etres.", channel}}

 }
 // note how we return a pointer to DataFetched because it implements Fetcher (the return type of this func)
 return &DataFetched{items: items, err: err}
}

// Remember that the Subscribe func call this func in a goroutine. The func does the actual fetching of updates and channeling them
// back to the calling funcs. There are a few cool construct here that are hard to implement in other languages (Java for instance)
func (s *Subscription) serve() {
 serviceId := r.Int63()
 fmt.Println("starting service", serviceId)
 // stagger fetching so no more that maxPending is queued
 const maxPending = 5
 var (
  next          time.Time
  pendingData   []Item
  dupFilter     = make(map[string]bool)
  firstItem     Item
  fetchingBatch chan *DataFetched
 )
 // loop until told to close
 for {
  //we insure not exceeding maxPending by having a nil chan that blocks
  var nextFetch <-chan time.Time

  // we want to fetch only if the current fetch has done and the pending updates less that max
  if fetchingBatch == nil && len(pendingData) < maxPending {
   nextFetch = time.After(next.Sub(time.Now()))
  }
  // note how the "updates" chan is defined only if there are pedning data. The data is cosumed one item 
  // at a time from the feching chan
  var updates chan Item
  if len(pendingData) > 0 {
   updates = s.updates
   // at every loop read one item from the pending data
   firstItem = pendingData[0]
  }
  select {
  case <-nextFetch:
   //we don't want to block on fetch so the pending queue keeps on draining and close signals are received. 
   // But at the same time we don't want to issue a fetch until the older fetch has finished
   fetchingBatch = make(chan *DataFetched)
   go func() {
    var (
     items []Item
     err   error
    )
    fmt.Println("Herr Bismark wird abrufen @", next)
    items, next, err = s.fetcher.fetch()
    // send the fetched data on the chan fetchingBach which will be selected further down
    fetchingBatch <- &DataFetched{items, next, err}
   }()

  //send the first item in the pending data read above on the updates chan. Nil channels block
  case updates <- firstItem:
   // slice the pending data so the first item is gone
   pendingData = pendingData[1:]
  case echan := <-s.closeit:
   echan <- fmt.Errorf("%s: %d", "breaking the service loop for service", serviceId)
   close(s.updates)
   return
  // the fetched batch that was send on the chan is consume here
  case df := <-fetchingBatch:
   fetchingBatch = nil
   if df.err != nil {
    fmt.Println("Error: Crap happens!")
    next = time.Now().Add(10 * time.Second)
   } else {
    // read the items one at a time from the fetched data and put it in the pendingData list
    // we filter out duplicate updates
    for _, i := range df.items {
     _, ok := dupFilter[i.ID]
     if !ok {
      pendingData = append(pendingData, i)
      dupFilter[i.ID] = true
     } else {
      fmt.Println("duplicate item")
     }
    }
   }

  }
 }
}

func main() {
 // Subscribe to some feeds, and create a merged update stream.
 merged := Merge(
  Subscribe(Fetch("blog.golang.org")),
  Subscribe(Fetch("googleblog.blogspot.com")),
  Subscribe(Fetch("googledevelopers.blogspot.com")))

 // Close the subscriptions after some time.
 time.AfterFunc(10*time.Second, func() {
  fmt.Println("closed:", merged.Close())
 })
 // Here's how we print the data that become available on the updates stream
 for it := range merged.UpdatesStream() {
  fmt.Println("Received... ", it.ID, it.Chan, it.Title)
 }
 // this should not show any go routines running which would constitute a leak
 panic("show me the stacks")
}
