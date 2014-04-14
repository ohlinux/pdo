package main

import (
    "flag"
    "fmt"
)

var workers int;

func main() {
flag.IntVar(&workers,"r", 1, "concurrent processing ,default 1 .")
    flag.Parse()
    test()
}

func test(){
    fmt.Println(workers)
}
