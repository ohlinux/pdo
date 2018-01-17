package pdo

import (
	"fmt"
	"io"
	"bufio"
	"strings"
	"os"
	"encoding/json"

	"gopkg.in/yaml.v2"
)

// 按行显示
func (pdo *Pdo)OutputFormatRow(host string , stdout io.Reader,regex string,tofile bool)error {
	scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			if strings.Contains(scanner.Text(), regex) {
				if tofile {
						pdo.OutputSaveToFile(fmt.Sprintf("[%s] %s\n",host,scanner.Text()))
				}else{
					fmt.Printf(">> \033[34m%-25s\033[0m >> %s\n", host, strings.Replace(scanner.Text(),regex, "\033[1;31m"+regex+"\033[0m", -1))
				}
			} else {
				if tofile {
					pdo.OutputSaveToFile(fmt.Sprintf("[%s] %s\n",host,scanner.Text()))
				}else{
					fmt.Printf(">> \033[34m%-25s\033[0m >> %s\n", host, scanner.Text())
				}
			}
		}

	return scanner.Err()

}


func (pdo *Pdo)OutputFormatJson(result JobResult){
	fmt.Println("------")
	str,err:=json.MarshalIndent(result,""," ")
	if err !=nil {
		fmt.Println(err)
	}else{
		fmt.Println(string(str))
	}
}


func (pdo *Pdo)OutputFormatYaml(result JobResult){
	fmt.Println("------")
	str,err:=yaml.Marshal(result)
	if err !=nil {
		fmt.Println(err)
	}else{
		fmt.Println(string(str))
	}
}


func (pdo *Pdo)OutputSaveToDirectory(host ,directory string,jobid int , stdout,stderr io.Reader )error {
	outFile := fmt.Sprintf("%s/%s_%d", directory, host,jobid)
	outf, err := os.Create(outFile)
	//defer outf.Close()
	go io.Copy(outf, stdout)
	go io.Copy(outf, stderr)
	return err
}

func (pdo *Pdo)OutputSaveToFile(line string)error {
	_, err := pdo.Output.File.WriteString(line)
	//todo: debug
	return err
}

