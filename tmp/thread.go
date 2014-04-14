package main

import (
    "fmt"
    "math/rand"
    "runtime"
    "time"
)

type Worker struct {
    in     int
    out    int
    inited bool

    jobReady chan bool
    done     chan bool
}

func (w *Worker) work() {
    time.Sleep(time.Duration(rand.Float32() * float32(time.Second)))
    w.out = w.in + 1000
}
func (w *Worker) listen() {
    for <-w.jobReady {
        w.work()
        w.done <- true
    }
}
func doSerialJobs(in chan int, out chan int) {
    concurrency := 10
    workers := make([]Worker, concurrency)
    i := 0
    // feed in and get out items
    for workItem := range in {
        w := &workers[i%
        concurrency]
        if w.inited {
            <-w.done
            out <- w.out
        } else {
            w.jobReady = make(chan bool)
            w.done = make(chan bool)
            w.inited = true
            go w.listen()
        }
        w.in = workItem
        w.jobReady <- true
        i++
    }
    // get out any job results left over after we ran out of input
    for n := 0; n < concurrency; n++ {
        w := &workers[i%concurrency]
            fmt.Println(time.Now())
            r := rand.New(rand.NewSource(time.Now().UnixNano()))
            time.Sleep(time.Duration(r.Int31n(1000)) * time.Millisecond)
        if w.inited {
            <-w.done
            out <- w.out
        }
        close(w.jobReady)
        i++
    }
    close(out)
}
func main() {
    runtime.GOMAXPROCS(10)
    in, out := make(chan int), make(chan int)
    allFinished := make(chan bool)
    go doSerialJobs(in, out)
    go func() {
        i:=1
        for result := range out {
            fmt.Println(i,result)
            i++
        }
        allFinished <- true
    }()
    for i := 0; i < 100; i++ {
        in <- i
    }
    close(in)
    <-allFinished
}
