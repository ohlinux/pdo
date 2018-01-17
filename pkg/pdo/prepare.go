package pdo

import (
	"fmt"
	"io"
	"bufio"
	"strings"
	"os"
	"net/url"
	"regexp"

	"github.com/spf13/cobra"
)

func (pdo *Pdo)PrepareInput(cmd *cobra.Command, args []string) {


	incmd:=make(map[string]string)
	ins:=[]string{"in-format","in-file","in-last-fail","in-last-success","in-regex"}
	for _,v:=range ins {
		incmd[v]=cmd.PersistentFlags().Lookup(v).Value.String()
	}


	pdo.Input.Format=incmd["in-format"]
	pdo.Input.Regex=incmd["in-regex"]

	if incmd["in-last-fail"] == "true" {
		pdo.Input.From=FAILF
	}else if incmd["in-last-success"] == "true"{
		pdo.Input.From=SUCCESSF
	}else if incmd["in-file"] != "" {
		pdo.Input.From=incmd["in-file"]
	}

}

func (pdo *Pdo)PrepareOutput(cmd *cobra.Command, args []string) {

	var f *os.File
	var err error

	outcmd := make(map[string]string)
	outs := []string{"out","out-directory", "out-file", "out-file-append", "out-regex"}
	for _, v := range outs {
		fmt.Println(v)
		outcmd[v] = cmd.PersistentFlags().Lookup(v).Value.String()
	}

	pdo.Output.Regex = outcmd["out-regex"]
	pdo.Output.Format = outcmd["out"]

	pdo.Output.Save=make(map[string]string)

	if outcmd["out-directory"] != "" {
		pdo.Output.Save[OutDirectory]=outcmd["out-directory"]
			if err:=os.MkdirAll(outcmd["out-directory"],0777) ; err!=nil {
				fmt.Errorf("create the output directory %s fail %v",outcmd["out-directory"],err)
			}
	}else if outcmd["out-file"] != "" {
		pdo.Output.Format="row"
		pdo.Output.Save[OutFile] = outcmd["out-file"]

		f,err=os.Create(outcmd["out-file"])
		if err!=nil {
			fmt.Println("create file failed,",err)
			os.Exit(1)
		}

	}else if outcmd["out-file-append"]  != "" {
		pdo.Output.Format="row"
		pdo.Output.Save[OutFileAppend] = outcmd["out-file-append"]

		f, err = os.OpenFile(outcmd["out-file-append"], os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			fmt.Println("create file failed,",err)
			os.Exit(1)
		}
	}

	pdo.Output.File=f
}


//创建host列表  Input来源 文件或者管道
func (pdo *Pdo)CreateJobList() error {

	var source io.Reader
	var err error
	if pdo.Input.From != "" {
		source, err = os.Open(pdo.Input.From)
		if err !=nil {
			return  err
		}
	}else{
		source=os.Stdin
	}

	var list HostList
	var lists []HostList
	sshPrefix:="ssh://"

	scanner := bufio.NewScanner(source)
	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		if len(words) >= 1 {
			var path string
			requestUrl:=words[0]
			fmt.Println(requestUrl)

			if len(words)==2 {
				path=words[1]
			}else{
				path=HOME
			}

			if ! strings.HasPrefix(requestUrl,sshPrefix) {
				requestUrl=sshPrefix+requestUrl
			}
			u, err := url.Parse(requestUrl)
			if err != nil {
				return err
			}

			if u.Path != "" {
				path=u.Path
			}

			list.Path=path

			if u.Host == ""{
				return fmt.Errorf("input host list error %+v\n",requestUrl)
			}else{
				n:=strings.Split(u.Host,":")
				list.Host = n[0]
				if len(n)>=2 {
					list.Port = n[1]
				}
			}

			if u.User != nil {
				list.User=u.User.Username()
				if pass,ok:=u.User.Password();ok{
					list.Passwd=pass
				}
			}

			if list.User==""{
				if pdo.Auth.User != ""{
					list.User=pdo.Auth.User
				}else{
					list.User=USERNAME
				}
			}

			if list.Passwd == ""{
				if pdo.Auth.Passwd != "" {
					list.Passwd=pdo.Auth.Passwd
				}
			}

			if u.Port() == "" {
				list.Port=DefaultSSHPort
			}

		}

		if pdo.FilterHosts(list.Host) && pdo.UniqHosts(lists, list) {
			lists = append(lists, list)
		}

		//lists=append(lists,list)
	}
	err = scanner.Err()
	pdo.JobList=lists
	pdo.Jobsinfo.Total=len(lists)
	fmt.Println(pdo.JobList)
	return err
}

//匹配Host
func (pdo *Pdo)FilterHosts(host string) bool {
	matchd,err:= regexp.Match(pdo.Input.Regex,[]byte(host))
	if err !=nil {
		return true
	}
	return matchd
}

//去重
func (pdo *Pdo)UniqHosts(lists []HostList, list HostList) bool {
	for _, va := range lists {
		if va == list {
			return false
		}
	}
	return true
}

func (pdo *Pdo)PrepareWorkEnv()error {
	// create temp directory
	if err := os.MkdirAll(USERTMPDIR, 0777); err !=nil {
		return err
	}

	if f,err:=os.Create(FAILF); err!=nil {
		return err
	}else{
		pdo.WorkEnv.FailFile=f
	}

	if f,err:=os.Create(SUCCESSF); err!=nil {
	return err
	}else{
		pdo.WorkEnv.SuccessFile=f
	}
	return nil
}
