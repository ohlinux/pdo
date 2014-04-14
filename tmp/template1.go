package main

import (
    "os"
    "text/template"
)

func main() {
    muban1 := `hi, {{template "M2"}},
hi, {{template "M3"}}
`
    muban2 := `我是模板2，{{template "M3"}}`
    muban3 := "ha我是模板3ha!"

    tmpl, err := template.New("M1").Parse(muban1)
    if err != nil {
            panic(err)
    }   
    tmpl.New("M2").Parse(muban2)
    if err != nil {
            panic(err)
    }   
    tmpl.New("M3").Parse(muban3)
    if err != nil {
            panic(err)
    }   
    err = tmpl.Execute(os.Stdout, nil)
    if err != nil {
            panic(err)
    }   
}
