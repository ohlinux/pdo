package main

import (
    "fmt"
    "flag"
    "strings"
)

var (

    conf=flag.String("c","","configure")

)

func main(){
    flag.Parse()
    fmt.Println(*conf)
    cmdline:=strings.Join(flag.Args(),"\"")
    fmt.Println(cmdline)
}
