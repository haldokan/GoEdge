package main

import (
 "errors"
 "fmt"
 "math/rand"
 "strconv"
 "time"
)

// can also simply use rand.Seed(int64)
var r *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

type Item struct{ ID, Title, Chan string }

type Fetcher interface {
 fetch() (items []Item, next time.Time, err error)
}

type DataFetched struct {
 items []Item
 next  time.Time
 err   error
}

type Subscriber interface {
 UpdatesStream() <-chan Item
 Close() error
 ClosingChan() chan chan error
}

type Subscription struct {
 fetcher Fetcher
 updates chan Item
 closeit chan chan error
}

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

func Subscribe(fetcher Fetcher) Subscriber {
 s := &Subscription{
  fetcher: fetcher, updates: make(chan Item), closeit: make(chan chan error),
 }
 go s.serve()
 return s
}

//going to assume 3 subscriptions to avoid using reflection (func Select(cases []SelectCase) (chosen int, recv Value, recvOK bool))
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

func (d *DataFetched) fetch() (items []Item, next time.Time, err error) {
 d.next = time.Now().Add(time.Duration(2 * time.Second))
 return d.items, d.next, d.err
}

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
 return &DataFetched{items: items, err: err}

}

// serve fetches items using s.fetcher and sends them
// on s.updates.  serve exits when s.Close is called.
func (s *Subscription) serve() {
 serviceId := r.Int63()
 fmt.Println("starting service", serviceId)
 const maxPending = 5
 var (
  next          time.Time
  pendingData   []Item
  dupFilter     = make(map[string]bool)
  firstItem     Item
  fetchingBatch chan *DataFetched
 )
 for {
  //we insure not exceeding maxPending by having a nil channel that blocks
  var nextFetch <-chan time.Time

  if fetchingBatch == nil && len(pendingData) < maxPending {
   // create a chan only if the pending queue is ok
   nextFetch = time.After(next.Sub(time.Now()))
  }
  var updates chan Item
  if len(pendingData) > 0 {
   updates = s.updates
   firstItem = pendingData[0]
  }
  select {
  case <-nextFetch:
   //we don't want to block on fetch so the pending queue keeps on draining and close signals are received. But at the same time we don't want to issue a fetch until the older
   // fetch has finished
   fetchingBatch = make(chan *DataFetched)
   go func() {
    var (
     items []Item
     err   error
    )
    fmt.Println("Herr Bismark wird abrufen @", next)
    items, next, err = s.fetcher.fetch()
    fetchingBatch <- &DataFetched{items, next, err}
   }()

  //nil channels block
  case updates <- firstItem:
   pendingData = pendingData[1:]
  case echan := <-s.closeit:
   echan <- fmt.Errorf("%s: %d", "breaking the service loop for service", serviceId)
   close(s.updates)
   return
  case df := <-fetchingBatch:
   fetchingBatch = nil
   if df.err != nil {
    fmt.Println("Error: Shit happens!")
    next = time.Now().Add(10 * time.Second)
   } else {
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
 // Print the stream.
 for it := range merged.UpdatesStream() {
  fmt.Println("Received... ", it.ID, it.Chan, it.Title)
 }
 // this should not show any go routines running which would constitute a leak
 panic("show me the stacks")
}
