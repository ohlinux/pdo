package main

import (
    "os"
    "text/template"
)

type Inventory struct {
    Material string
    Count    uint
}

func main() {
    sweaters := Inventory{"wool", 17} 
    muban := "{{.Count}} items are made of {{.Material}}"
    tmpl, err := template.New("test").Parse(muban)  //建立一个模板
    if err != nil {   
            panic(err)
    }   
    err = tmpl.Execute(os.Stdout, sweaters) //将struct与模板合成，合成结果放到os.Stdout里
    if err != nil {
            panic(err)
    }   
}
//输出 ：   17 items are made of wool
