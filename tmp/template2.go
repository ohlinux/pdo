package main

import (
    "os"
    "text/template"
)

func main() {
    name := "waynehu"
    tmpl, err := template.New("test").Parse("hello, {{.}}") //建立一个模板，内容是"hello, {{.}}"
    if err != nil {   
            panic(err)
    }   
    err = tmpl.Execute(os.Stdout, name)  //将string与模板合成，变量name的内容会替换掉{{.}} 
    //合成结果放到os.Stdout里
    if err != nil {
            panic(err)
    }   
}
//输出 ：   hello, waynehu
