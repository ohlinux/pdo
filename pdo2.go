package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"io"
	//    "io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"strconv"
	"strings"
	//"sync"
	"text/template"
	//"math/rand"
	"database/sql"
	log "github.com/cihub/seelog"
	_ "github.com/go-sql-driver/mysql"
	"github.com/robfig/config"
	"runtime"
	"time"
)

const AppVersion = "Version 2.0.20140803"

var (
	SUDO_USER  = os.Getenv("SUDO_USER")
	USERNAME   = os.Getenv("username")
	HOME       = os.Getenv("HOME")
	PID        = strconv.Itoa(os.Getppid())
	workers    = flag.Int("r", 1, "concurrent processing ,default 1 .")
	waitTime   = flag.Duration("t", 300*time.Second, "over the time ,kill process.")
	interTime  = flag.Duration("T", 0*time.Second, "Interval time between concurrent programs.")
	hostFile   = flag.String("f", "", "host list,allow 1 or 2 columns. the first column is HostName or IP , the second column is path.")
	outputDir  = flag.String("o", "", "output dir. ")
	outputShow = flag.String("show", "", "show option,you can use <row>")
	product    = flag.String("p", "", "input product name.")
	app        = flag.String("a", "", "input app name.")
	mstring    = flag.String("match", "", "match a string with color.")
	mrule      = flag.String("rule", "", "rule for match string in conf.")
	//    argsConf   = flag.String("c", "", "args conf,save to file.")
	//copy       = flag.String("c", "", "Copy <file> to destination as <dest_path>. If notspecified, <dest_path> will be same as <file>.")
	//script     = flag.String("e", "", "Transfer/execute a script from the local system to eachtarget system.")
	idc  = flag.String("i", "", "input filter idc name.")
	idcs = flag.String("I", "", "filter JX or TC .")
	//shortCmd   = flag.String("cmd", "", "short command in conf.")
	configFile = flag.String("C", HOME+"/.pdo/pdo.conf", "configure file ,default /tmp/pdo_conf.$$.")
	yesorno    = flag.Bool("y", false, "continue, yes/no , default false.")
	retry      = flag.Bool("R", false, "retry fail list at the last time")
	quiet      = flag.Bool("q", false, "quiet mode,disable header output.")
	scriptTemp = flag.String("temp", "", "use template")
	builtin    = flag.String("b", "", "built-in for template,when use temp")

	usage = `
  input control:
    -f <file>           from File "HOST PATH".
    -a <orp appname>    from database.
    -p <orp product>    from database.
    -R                  from last failure list.
    default             from pipe,eg: cat file | pdo2

  output control:
    default             display after finish.
    -show <row>         display line by line.
    -o <dir>            save to directory.

  thread control:
    -r <int>            concurrent processing ,default 1.
    -t <10s>            over the time ,kill process.
    -T <1m>             Interval time between concurrent programs.
    -y                  default need input y .
    -q                  quiet mode , not display the head infomation.

  subcommand :
    copy <file> <destination>   copy file to remote host.
    script <script file>        execute script file on remote host.
    cmd <config shortcmd>       use shortcmd in the pdo.conf.
    tail <file>                 mulit tail -f file .
    md5sum <file>               get md5sum file and count md5.[not finished]
    help <subcommand>           get subcommand help.
    version                     get the pdo version.
    setup                       setup the configuration at the first time .[not finished]
    conf                        save the used args.[not finished]

  Examples:
  ##simple ,read from pipe.
    cat list | pdo2 "pwd"
  ##-a from orp , -r  concurrent processing
    pdo2 -a download-client -r 10 "pwd"
  ##-show row ,show line by line
    pdo2 -p tieba -y -show row "pwd"
  ##copy files
    pdo2 -a download-client copy 1.txt /tmp/
  ##excute script files
    pdo2 -a download-client script test.sh
  ## local command
    pdo2 -a download-client "scp a.txt {{.Host}}:{{.Path}}/log/"
  ##more help
    <http://tiebaop.baidu.com/docs/scripts/pdo/>
`
)

type Pdo struct {
	Concurrent    int
	JobList       []HostList
	JobTotal      int
	JobFinished   int
	CmdLastString string
	PdoLocal      bool
	FailFile      string
	MatchString   string
	QuietOption   bool
	OutputWay     string
	TimeWait      time.Duration
	TimeInterval  time.Duration
	SubCommand    string
	OutputDir     string
	TemplateFile  string
	YesOrNo       bool
	//	ScriptFile    string
	//	CopyFile      string
	Jobs  Job
	Pause bool
}

type Job struct {
	jobname HostList
	results chan<- Result
}

type Result struct {
	jobname    string
	resultcode int
	resultinfo string
}

type HostList struct {
	Host string
	Path string
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	//获取相关信息
	pdo := NewPdo()

	//头部显示
	pdo.displayHead()

	YesNO(pdo.YesOrNo)

	//并发调度处理
	pdo.doRequest()

	//尾部显示
	pdo.displayEnd()
}

//#################
//主体main的相关函数
//获取和判断相关信息
func NewPdo() *Pdo {

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [input control][thread control] [output control][subcommand] <content>\n", os.Args[0])
		fmt.Fprint(os.Stderr, usage)
		//flag.PrintDefaults()
	}
	//参数解析
	flag.Parse()

	var info *Pdo

	//    argsFile:="/tmp/pdo_conf."+PID

	//    if *argsConf != "" {

	//        info=parseArgsConf(*argsConf)
	//
	//    }else if  checkExist(argsFile){
	//
	//        info=parseArgsConf(argsFile)
	//
	//    }else{

	info = &Pdo{
		Concurrent:   *workers,
		OutputDir:    *outputDir,
		FailFile:     "/tmp/pdo_faile." + PID,
		MatchString:  *mstring,
		OutputWay:    *outputShow,
		QuietOption:  *quiet,
		TimeWait:     *waitTime,
		TimeInterval: *interTime,
		YesOrNo:      *yesorno,
	}

	//限制rd的账号
	if USERNAME == "rd" {
		info.Concurrent = 50
	}

	//   }

	//PdoLocal      : templateTrue,
	//CmdLastString : tempCommand ,

	//subcommand处理
	var preCommand, tempCommand, subCommand string
	if len(flag.Args()) > 0 {
		preCommand = flag.Args()[0]
	} else {
		fmt.Println("Input Error,use help.")
		os.Exit(1)
	}
	switch preCommand {
	case "version":
		fmt.Println(AppVersion)
		os.Exit(0)
	case "help":
		//帮助
		pdoHelp(flag.Args()[1])
	case "setup":
		//第一次创建.
		if pdoSetup() {
			fmt.Println("SETUP SUCCESS.")
			fmt.Println("Please Check the ~/.pdo/pdo.conf and copy the bin to $PATH ...")
			os.Exit(0)
		} else {
			fmt.Println("SETUP FAIL.")
			os.Exit(1)
		}
	case "conf":
		//参数配置
		//saveConf(info)
	case "copy":
		subCommand = preCommand
		tempCommand = flag.Args()[1] + "<:::>" + flag.Args()[2]
	case "cmd":
		subCommand = "bash"
		conf, _ := config.ReadDefault(*configFile)
		tempCommand, _ = conf.String("CMD", flag.Args()[1])
	case "tail":
		subCommand = "bash"
		tempCommand = "tail -f " + flag.Args()[1]
		info.OutputWay = "row"
		info.YesOrNo = true
	case "md5sum":
		subCommand = preCommand
		tempCommand = flag.Args()[1]
	case "script":
		//脚本执行
		subCommand = preCommand
		tempCommand = flag.Args()[1] + "<:::>" + "/tmp/pdo_script." + PID
	default:
		//默认执行.
		subCommand = "bash"
		tempCommand = preCommand
	}

	info.SubCommand = subCommand
	info.CmdLastString = tempCommand

	//配置文件
	conf, err := config.ReadDefault(*configFile)
	checkErr(1, err)

	//日志记录
	logConf, _ := conf.String("PDO", "logconf")
	logger, _ := log.LoggerFromConfigAsFile(logConf)
	log.ReplaceLogger(logger)

	//记录输入状态
	log.Info(SUDO_USER + " " + PID + " Do: " + strings.Join(os.Args, " "))
	log.Flush()

	//命令行本地命令与远程命令的处理判断.
	info.PdoLocal = false
	if strings.Contains(tempCommand, "{{.}}") || strings.Contains(tempCommand, "{{.Host}}") || strings.Contains(tempCommand, "{{.Path}}") {
		info.PdoLocal = true
	}

	//数据库配置
	var conn string
	if *product != "" || *app != "" {
		dbhost, err := conf.String("Mysql", "Host")
		dbport, err := conf.String("Mysql", "Port")
		dbuser, err := conf.String("Mysql", "User")
		dbpass, err := conf.String("Mysql", "Pass")
		dbname, err := conf.String("Mysql", "DBname")
		conn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbuser, dbpass, dbhost, dbport, dbname)
		checkErr(1, err)
	}
	//input判断
	if *product != "" {
		info.JobList, _ = ListProductMysql(conn, *product)
	} else if *app != "" {
		info.JobList, _ = ListAppMysql(conn, *app)
	} else if *hostFile != "" {
		sourceFile, err := os.Open(*hostFile)
		checkErr(1, err)
		info.JobList, _ = CreateHostList(sourceFile)
	} else if *retry {
		sourceFile, err := os.Open(info.FailFile)
		checkErr(1, err)
		info.JobList, _ = CreateHostList(sourceFile)
	} else {
		info.JobList, _ = CreateHostList(os.Stdin)
	}

	//对jobList进行优化处理.

	info.JobTotal = len(info.JobList)

	log.Info(info.JobList)
	defer log.Flush()

	//没有输入命令
	if info.CmdLastString == "" {
		fmt.Println("[ Error ] cmd input error ,no command or action need to do ! exit...")
		os.Exit(1)
	}

	return info

}

//头部信息显示显示
func (pdo *Pdo) displayHead() {
	//头部输出
	if !pdo.QuietOption {
		fmt.Println(">>>> Welcome " + SUDO_USER + "...")
		for i, elem := range pdo.JobList {
			fmt.Printf("%-25s-%-20s ", elem.Host, elem.Path)
			if (i+1)%2 == 0 {
				fmt.Printf("\n")
			}
		}
		fmt.Printf("\n")
		fmt.Println("#--Total--# ", pdo.JobTotal)

		switch pdo.SubCommand {
		case "copy":
			cmdLine := strings.Split(pdo.CmdLastString, "<:::>")
			fmt.Println("#---CMD---#  Copy:", cmdLine[0], "-->", cmdLine[1])
		case "script":
			cmdLine := strings.Split(pdo.CmdLastString, "<:::>")
			fmt.Println("#---CMD---#  Script:", cmdLine[0])
		default:
			fmt.Println("#---CMD---# ", pdo.CmdLastString)

		}
	}

	if pdo.JobTotal == 0 {
		fmt.Println("No Host List !Please check input . exit...")
		os.Exit(1)
	}

	//create fail list
	err := os.MkdirAll("/tmp/pdo", 0777)
	checkErr(2, err)
	os.Create(pdo.FailFile)

	//mkdir  output
	if pdo.OutputDir != "" {
		err := os.MkdirAll(pdo.OutputDir, 0755)
		checkErr(2, err)
	}

}

//尾部信息显示
func (pdo *Pdo) displayEnd() {

}

//####################
//并发调度过程
//处理job对列
//并发调度开始
func (pdo *Pdo) doRequest() {
	jobs := make(chan Job, pdo.Concurrent)
	results := make(chan Result, len(pdo.JobList))
	done := make(chan struct{}, pdo.Concurrent)

	go pdo.addJob(jobs, pdo.JobList, results)

	for i := 0; i < pdo.Concurrent; i++ {
		go pdo.doJob(done, jobs)
	}

	go pdo.sysSignalHandle()

	go pdo.awaitCompletion(done, results, pdo.Concurrent)

	pdo.processResults(results)
}

//添加job
func (pdo *Pdo) addJob(jobs chan<- Job, jobnames []HostList, results chan<- Result) {
	for num, jobname := range jobnames {
		jobs <- Job{jobname, results}
		//第一个任务暂停
		if !pdo.QuietOption && num == 0 {
			for {
				if pdo.Pause {
					YesNO(pdo.YesOrNo)
					break
				} else {
					time.Sleep(10 * time.Millisecond)
				}
			}
		}
	}
	close(jobs)
}

//处理job
func (pdo *Pdo) doJob(done chan<- struct{}, jobs <-chan Job) {

	for job := range jobs {
		pdo.Do(&job)
		time.Sleep(pdo.TimeInterval)
	}
	done <- struct{}{}
}

//job完成状态
func (pdo *Pdo) awaitCompletion(done <-chan struct{}, results chan Result, works int) {
	for i := 0; i < works; i++ {
		<-done
	}
	close(results)
}

//job处理结果
func (pdo *Pdo) processResults(results <-chan Result) {
	//0 success
	//1 fail
	//2 time over killed
	//3 time over kill failed

	jobnum := 1
	success := 0
	fail := 0
	overtime := 0
	for result := range results {
		switch result.resultcode {
		case 0:
			if pdo.OutputWay == "" {
				fmt.Printf("[%d/%d] %s \033[34m [SUCCESS]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			}
			success++
		case 2:
			fmt.Printf("[%d/%d] %s \033[1;31m [Time Over KILLED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			CreateFailFile(pdo.FailFile, result.jobname)
			overtime++
		case 3:
			fmt.Printf("[%d/%d] %s \033[1;31m [KILLED FAILED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			CreateFailFile(pdo.FailFile, result.jobname)
			overtime++
		default:
			if pdo.OutputWay == "" && pdo.OutputDir == "" {
				fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			}
			CreateFailFile(pdo.FailFile, result.jobname)
			fail++
		}
		fmt.Println(result.resultinfo)
		jobnum++
	}
	fmt.Printf("[INFO] Total: %d ; Success: %d ; Failed: %d ; OverTime: %d\n", pdo.JobTotal, success, fail, overtime)
}

//####################
//公共调用函数
//错误检查
func checkErr(i int, err error) {
	if err != nil {
		switch i {
		case 1:
			log.Critical(err)
		case 2:
			log.Warn(err)
		default:
			log.Info(err)
		}
	}
	log.Flush()
}

//检查文件是否存在.
func checkExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

//过滤Host
func FilterHosts(hostName string) bool {

	filterString := *idc
	if *idcs != "" {
		c, _ := config.ReadDefault(*configFile)
		filterString, _ = c.String("IDC", *idcs)
	}

	sidc := strings.Split(hostName, ".")
	if filterString != "" {
		idc := strings.Split(filterString, ",")
		for _, va := range idc {
			if va == sidc[1] {
				return false
			}
		}
	}
	return true
}

//去重Host+Path
func UniqHosts(lists []HostList, list HostList) bool {
	for _, va := range lists {
		if va == list {
			return false
		}
	}
	return true
}

//判断Yes or No的输入
func YesNO(choose bool) {
	userFile := "/dev/tty"
	if !choose {
	here:
		fmt.Printf("Continue (y/n):")
		fin, err := os.Open(userFile)
		defer fin.Close()
		if err != nil {
			fmt.Println(userFile, err)
			return
		}
		buf := make([]byte, 1024)
		for {
			n, _ := fin.Read(buf)
			if 0 == n || n > 1 {
				break
			}
		}
		switch string(buf[:2]) {
		case "y\n":
			fmt.Println("go on ...")
		case "n\n":
			fmt.Println("exit ...")
			os.Exit(1)
		default:
			goto here
		}
	}
}

//创建host列表  Input来源 文件或者管道
func CreateHostList(file *os.File) ([]HostList, error) {
	var list HostList
	var lists []HostList

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		words := strings.Fields(scanner.Text())
		if len(words) >= 1 {
			if len(words) < 2 {
				list.Host = words[0]
				list.Path = HOME
			} else {
				list.Host = words[0]
				list.Path = words[1]
			}
			if FilterHosts(list.Host) && UniqHosts(lists, list) {
				lists = append(lists, list)
			}
		}
	}
	err := scanner.Err()
	checkErr(2, err)
	return lists, err
}

//创建失败文件
func CreateFailFile(failFile string, contents string) error {
	file, err := os.OpenFile(failFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	defer file.Close()
	checkErr(2, err)
	file.WriteString(contents + "\n")
	return err
}

//信号处理
func (pdo *Pdo) sysSignalHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Warnf("Ctrl+c,recode fail list to "+pdo.FailFile+" ,signal:%s", sig)
			for x := pdo.JobFinished - 1; x < pdo.JobTotal; x++ {
				CreateFailFile(pdo.FailFile, pdo.JobList[x].Host+" "+pdo.JobList[x].Path)
			}
			os.Exit(0)
		}
	}()
}

//具体job处理过程
func (pdo *Pdo) Do(job *Job) {

	pdo.JobFinished++
	var out, outerr bytes.Buffer
	var cmd *exec.Cmd

	fmt.Println(pdo.SubCommand)
	switch pdo.SubCommand {
	case "copy":
		//copy文件
		cmdLine := strings.Split(pdo.CmdLastString, "<:::>")
		ch := cmdLine[1][0]
		if string(ch) == "/" {
			cmd = exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", cmdLine[0], job.jobname.Host+":"+cmdLine[1])
		} else {
			cmd = exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", cmdLine[0], job.jobname.Host+":"+job.jobname.Path+"/"+cmdLine[1])
		}
	case "script":
		//脚本执行 先copy 后 执行
		cmdLine := strings.Split(pdo.CmdLastString, "<:::>")
		remoteCmd := fmt.Sprintf("chmod +x %s && %s && cd /tmp/ && rm -f %s", cmdLine[1], cmdLine[1], cmdLine[1])
		copycmd := exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", cmdLine[0], job.jobname.Host+":"+cmdLine[1])
		copycmd.Run()
		cmd = exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", job.jobname.Host, "cd", job.jobname.Path, "&&", remoteCmd)
	default:
		//默认是命令行执行
		//本地脚本执行与远程执行
		if pdo.PdoLocal {
			tmpl, err := template.New("pdo").Parse(pdo.CmdLastString)
			checkErr(2, err)
			hostTemp := HostList{job.jobname.Host, job.jobname.Path}
			var tempBuf bytes.Buffer
			err = tmpl.Execute(&tempBuf, hostTemp)
			checkErr(2, err)
			outputString := tempBuf.String()
			cmd = exec.Command("/bin/bash", "-s")
			cmd.Stdin = bytes.NewBufferString(outputString)
		} else {
			cmd = exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", job.jobname.Host, "cd", job.jobname.Path, "&&", pdo.CmdLastString)
			//cmd:=exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", "job.jobname.Host","sh -s")
		}
	}

	//命令执行
	stdout, err := cmd.StdoutPipe()
	checkErr(2, err)
	stderr, err := cmd.StderrPipe()
	checkErr(2, err)
	err = cmd.Start()
	checkErr(2, err)

	//执行过程内容
	//输出存储到文件
	if pdo.OutputDir != "" {
		//写重复host时候
		i := 1
		outFile := fmt.Sprintf("%s/%s", pdo.OutputDir, job.jobname.Host)
		writeFile := outFile
		for {
			if checkExist(writeFile) {
				writeFile = outFile + "_" + strconv.Itoa(i)
			} else {
				break
			}
			i++
		}
		outf, err := os.Create(writeFile)
		defer outf.Close()
		checkErr(2, err)
		go io.Copy(outf, stdout)
		go io.Copy(outf, stderr)
	} else if pdo.OutputWay == "row" {
		//行输出显示
		scanner := bufio.NewScanner(stdout)
		if pdo.MatchString != "" {
			for scanner.Scan() {
				if strings.Contains(scanner.Text(), pdo.MatchString) {
					fmt.Printf(">> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, strings.Replace(scanner.Text(), pdo.MatchString, "\033[1;31m"+pdo.MatchString+"\033[0m", -1))
				} else {
					fmt.Printf(">> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, scanner.Text())
				}
			}
		} else {
			for scanner.Scan() {
				fmt.Printf(">> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil {
			checkErr(2, err)
		}
	} else {
		//直接输出
		go io.Copy(&out, stdout)
		go io.Copy(&outerr, stderr)
	}

	done := make(chan error)

	go func() {
		done <- cmd.Wait()
	}()

	jobstring := job.jobname.Host + " " + job.jobname.Path
	//线程控制执行时间
	select {
	case <-time.After(pdo.TimeWait):
		//超时被杀时
		if err := cmd.Process.Kill(); err != nil {
			//超时被杀失败
			job.results <- Result{jobstring, 3, "Killed..."}
			checkErr(2, err)
		}
		<-done
		job.results <- Result{jobstring, 2, "Time over ,Killed..."}
		//记录失败job
	case err := <-done:
		if err != nil {
			//完成返回失败
			job.results <- Result{jobstring, 1, outerr.String()}
		} else {
			//完成返回成功
			if pdo.OutputWay == "" {
				//如果是行显示就隐藏
				job.results <- Result{jobstring, 0, out.String()}
			}
		}
	}
	//判断如果是第一个job就暂停
	if pdo.JobFinished == 1 {
		pdo.Pause = true
	}
}

func pdoSetup() bool {

	confDir := HOME + "/.pdo"
	logDir := confDir + "/log"
	templateDir := confDir + "/template"

	conffile := confDir + "/pdo.conf"
	logfile := confDir + "/log.xml"

	err := os.MkdirAll(logDir, 0755)
	checkErr(1, err)
	err = os.MkdirAll(templateDir, 0755)
	checkErr(1, err)

	pdoconf := `
[PDO]
logconf:{{.}}/.pdo/log.xml
scripts:bash,python,ruby,perl,php

[DB]
Host:10.92.74.42
Port:5100
DBname:orp
User:orp_beiku
Pass:Ju38Jdfwluew

[IDC]
JX:yf01,cq01,dbl01,ai01,jx,cp01
TC:cq02,tc,m1,db01,st01
NJ:nj02

[TEMPLATE]
example : {{.}}/.pdo/template/example.sh

[CMD]
example : find -type d | wc -l

`

	logconf := `
<seelog minlevel="info">
    <outputs formatid="common">
        <rollingfile type="size" filename="{{.}}/.pdo/log/roll.log" maxsize="100000" maxrolls="5"/>
        <filter levels="critical">
            <file path="{{.}}/.pdo/log/critical.log" formatid="critical"/>
        </filter>
        <filter levels="warn">
            <console formatid="colored"/>
        </filter>
    </outputs>
    <formats>
        <format id="colored"  format="%Time %EscM(46)%Level%EscM(49) %Msg%n%EscM(0)"/>
        <format id="common" format="%Date/%Time [%LEV] %Msg%n" />
        <format id="critical" format="%File %FullPath %Func %Msg%n" />
        <format id="criticalemail" format="Critical error on our server!\n    %Time %Date %RelFile %Func %Msg \nSent by Seelog"/>
    </formats>
</seelog>
`

	if !checkExist(conffile) {
		fmt.Println(conffile + " not exist.")
		openconffile, err := os.OpenFile(conffile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		defer openconffile.Close()
		conftmpl, err := template.New("pdoconf").Parse(pdoconf)
		checkErr(1, err)
		err = conftmpl.Execute(openconffile, HOME)
		checkErr(1, err)

	}
	if !checkExist(logfile) {
		fmt.Println(logfile + " not exist.")
		openlogfile, err := os.OpenFile(logfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		defer openlogfile.Close()
		logtmpl, err := template.New("logconf").Parse(logconf)
		checkErr(1, err)
		err = logtmpl.Execute(openlogfile, HOME)
		checkErr(1, err)
	}

	if err == nil {
		return true
	} else {
		return false
	}

}

func pdoHelp(command string) {

}

//orp数据库源
func ListProductMysql(conn string, proName string) ([]HostList, error) {

	var Lists []HostList

	//connect to mysql
	db, err := sql.Open("mysql", conn)
	checkErr(1, err)
	defer db.Close()

	//sql
	stmtProductName, err := db.Prepare("select productId from product where productName=?")
	checkErr(2, err)

	stmtProductId, err := db.Prepare("select appName,appId  from appInfo where productId=? and status=1")
	checkErr(2, err)

	var proId, app, appId string
	err = stmtProductName.QueryRow(proName).Scan(&proId)
	checkErr(2, err)

	rows, err := stmtProductId.Query(proId)
	for rows.Next() {
		err = rows.Scan(&app, &appId)
		checkErr(2, err)
		arrApp, _ := ListAppMysql(conn, app)
		for _, va := range arrApp {
			Lists = append(Lists, va)
		}
	}

	return Lists, err
}

func ListAppMysql(conn string, appName string) ([]HostList, error) {

	var L HostList
	var Lists []HostList

	//connect to mysql
	db, err := sql.Open("mysql", conn)
	checkErr(1, err)
	defer db.Close()

	stmtAppName, err := db.Prepare("select appId  from appInfo where appName=? and status=1")
	checkErr(2, err)

	stmtRuntime, err := db.Prepare("select orpId,hostName,containerId from runtime where appId=?")
	checkErr(2, err)

	var appId string

	apps := strings.Split(appName, ",")
	for _, va := range apps {

		err = stmtAppName.QueryRow(va).Scan(&appId)
		checkErr(2, err)

		rows, err := stmtRuntime.Query(appId)
		for rows.Next() {
			var orpId, hostname, containerId string
			err = rows.Scan(&orpId, &hostname, &containerId)
			checkErr(2, err)
			if len(orpId) == 1 {
				L.Path = "/home/matrix/containers/" + containerId + "/home/work/orp00" + orpId
				//L.Path = "/home/work/orp00" + orpId
			} else {
				L.Path = "/home/matrix/containers/" + containerId + "/home/work/orp0" + orpId
				//L.Path = "/home/work/orp0" + orpId
			}

			L.Host = hostname
			//临时为新路径做过渡
			tmpFile, err := os.Open("/home/rd/duanbing/pdo_orp")
			checkErr(1, err)
			scanner := bufio.NewScanner(tmpFile)

			for scanner.Scan() {
				words := strings.Fields(scanner.Text())
				if len(words) == 2 {
					if words[0] == L.Host && words[1] == containerId {
						///home/matrix/containers/6.jx_pc_post_121/home/work/orp001
						L.Path = "/home/matrix/containers/" + containerId + "/home/work/orp"
						break
					}
				}
			}
			if FilterHosts(L.Host) && UniqHosts(Lists, L) {
				Lists = append(Lists, L)
			}
		}
	}
	return Lists, err
}
