package main

import (
	"bufio"
	"bytes"
	//"database/sql"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"text/template"
	"time"

	log "github.com/cihub/seelog"
	"github.com/robfig/config"
)

const AppVersion = "Version 2.0.20161017"

const (
	ResultSuccess        = 0
	ResultTimeOverKilled = 2
	ResultKillFailed     = 3
	ResultConnectFailed  = 255
)

// Version 2.0.20140821 增加print打印列表和prev提前显示的功能
// Version 2.0.20141121 fix bugs: -y 第一个job会先执行完才继续 ; ssh 输出的warning ; -o 输出文件冲突的问题; -o 失败不显示状态.
// Version 2.0.20151228 增加bns noahTree -b 的input
// Version 2.0.20151231 增加指定User 
// Version 2.0.20160130 retry -R 优先及最高. setup完善.
// Version 2.0.20160202 -bns -noah 增加多个的支持,中间用逗号分割.
// Version 2.0.20161007 增加connect failed的判断.
// Version 2.0.20161017 增加scrpit参数
var (
	SUDO_USER  = os.Getenv("SUDO_USER")
	USERNAME   = os.Getenv("USER")
	HOME       = os.Getenv("HOME")
	USERTMPDIR = "/tmp/pdo_" + USERNAME
	PID        = strconv.Itoa(os.Getppid())
	workers    = flag.Int("r", 1, "concurrent processing ,default 1 .")
	waitTime   = flag.Duration("t", 300*time.Second, "over the time ,kill process.")
	interTime  = flag.Duration("T", 0*time.Second, "Interval time between concurrent programs.")
	hostFile   = flag.String("f", "", "host list,allow 1 or 2 columns. the first column is HostName or IP , the second column is path.")
	outputDir  = flag.String("o", "", "output dir. ")
	outputShow = flag.String("show", "", "show option,you can use <row>")
	//product    = flag.String("p", "", "input product name.")
	//app        = flag.String("a", "", "input app name.")
	user       = flag.String("u", USERNAME, "remote user name.")
	//bns        = flag.String("bns", "", "input bns service name.")
	//noah       = flag.String("noah", "", "input noah tree.")
	mstring    = flag.String("match", "", "match a string with color.")
	mrule      = flag.String("rule", "", "rule for match string in conf.")
	grephost   = flag.String("host", "", "get the host list.")
	idc        = flag.String("i", "", "input filter idc name.")
	idcs       = flag.String("I", "", "filter JX or TC .")
	configFile = flag.String("C", HOME+"/.pdo/pdo.conf", "configure file ,default /tmp/pdo_conf.$$.")
	yesorno    = flag.Bool("y", false, "continue, yes/no , default false.")
	retry      = flag.Bool("R", false, "retry fail list at the last time")
	quiet      = flag.Bool("q", false, "quiet mode,disable header output.")
	scriptTemp = flag.String("temp", "", "use template")
	builtin    = flag.String("b", "", "built-in for template,when use temp")
	formula    = flag.String("formula", "", "-formula <object><formula><value>.")

	usage = `
  input control:
    -f <file>           from File "HOST PATH".
    -R                  from last failure list.
    default             from pipe,eg: cat file | pdo

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
    md5sum <file>               get md5sum file and count md5.
    help <subcommand>           get subcommand help.
    print                       print host list. hostname path.
    version                     get the pdo version.
    setup                       setup the configuration at the first time .[not finished]
    conf                        save the used args.[not finished]

  other :
    -u  <username>  	Specifies the remote user .

  Examples:
  ## the first time , setup pdo conf pdo.conf and log.xml
    pdo setup
  ## simple ,read from pipe.
    cat host.list | pdo "pwd"
  ## -show row ,show line by line
  ## copy files
    pdo -f host.list copy 1.txt /tmp/
  ## excute script files
    pdo -f host.list script test.sh
  ## local command
    pdo -f host.list "scp a.txt {{.Host}}:{{.Path}}/log/"
  ## Specifies the user
    pdo -u root -f host.list -r 10 "pwd"
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
	SuccessFile   string
	MatchString   string
	QuietOption   bool
	OutputWay     string
	TimeWait      time.Duration
	TimeInterval  time.Duration
	SubCommand    string
	OutputDir     string
	TemplateFile  string
	YesOrNo       bool
	Formula       string
	FormulaDir    string
	Jobs          Job
	Pause         bool
	FormulaArray  map[string]string
	User          string
}

type FormulaResult struct {
	Value string
	File  string
}

type Job struct {
	jobname HostList
	results chan<- Result
	jobid   int
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
		FailFile:     USERTMPDIR + "/faile." + PID,
		SuccessFile:  USERTMPDIR + "/success." + PID,
		MatchString:  *mstring,
		OutputWay:    *outputShow,
		QuietOption:  *quiet,
		TimeWait:     *waitTime,
		TimeInterval: *interTime,
		YesOrNo:      *yesorno,
		FormulaDir:   USERTMPDIR + "/formula_" + PID,
		User:         *user,
	}

	//限制rd的账号
	if USERNAME == "rd" {
		info.Concurrent = 50
	}

	//   }

	//PdoLocal      : templateTrue,
	//CmdLastString : tempCommand ,

	if *user == "root" {
		HOME = "/root"
	} else {
		HOME = "/home/" + *user
	}
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
			fmt.Println("[INFO] SETUP SUCCESS.")
			fmt.Println("       You can change the  conf ~/.pdo/pdo.conf and Please copy the pdo bin to $PATH .")
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
		info.OutputWay = "row"
		tempCommand = "md5sum " + flag.Args()[1]
		info.Formula = "1:diff"
	case "script":
		//脚本执行
		subCommand = preCommand
		tempCommand = flag.Args()[1] + "<:::>" + "/tmp/pdo_script." + PID + "<:::>" + strings.Join(flag.Args()[2:], " ")
	case "print":
		//打印机器列表
		subCommand = "print"
		tempCommand = "print"
		info.YesOrNo = true
		info.QuietOption = true
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

/*
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
*/
	//input判断
	if *retry {
		sourceFile, err := os.Open(info.FailFile)
		checkErr(1, err)
		info.JobList, _ = CreateHostList(sourceFile)
	} else if *hostFile != "" {
		sourceFile, err := os.Open(*hostFile)
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
			fmt.Println("#---CMD---#  Script:", cmdLine[0], cmdLine[2])
		default:
			fmt.Println("#---CMD---# ", pdo.CmdLastString)
		}
	}

	if pdo.SubCommand == "print" {
		for _, v := range pdo.JobList {
			fmt.Println(v.Host, v.Path)
		}
		os.Exit(0)

	}

	if pdo.JobTotal == 0 {
		fmt.Println("No Host List !Please check input . exit...")
		os.Exit(1)
	}

	//create fail list
	err := os.MkdirAll(USERTMPDIR, 0777)
	checkErr(2, err)
	os.Create(pdo.FailFile)
	os.Create(pdo.SuccessFile)

	//clear forumla dir
	if pdo.Formula != "" {
		os.RemoveAll(pdo.FormulaDir)
		//create fomula dir
		err = os.MkdirAll(pdo.FormulaDir, 0777)
		checkErr(2, err)
	}
	//mkdir  output
	if pdo.OutputDir != "" {
		err := os.MkdirAll(pdo.OutputDir, 0777)
		checkErr(2, err)
	}

}

//尾部信息显示
func (pdo *Pdo) displayEnd() {

	if pdo.Formula != "" {
		fmt.Println("Formula: " + pdo.Formula)

		for key, value := range pdo.FormulaArray {

			tmpFile, err := os.Open(pdo.FormulaDir + "/" + key)
			checkErr(1, err)
			scanner := bufio.NewScanner(tmpFile)
			num := 0
			for scanner.Scan() {
				num++
			}

			fmt.Println(key, num, value)
		}
	}

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
		jobs <- Job{jobname, results, num + 1}
		//第一个任务暂停
		if !pdo.QuietOption && num == 0 {
			for {
				if pdo.YesOrNo {
					break
				}
				if pdo.Pause {
					YesNO(pdo.YesOrNo)
					break
				} else {
					time.Sleep(20 * time.Millisecond)
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
	/*
		ResultSuccess = 0
		ResultTimeOverKILLED= 1
		ResultKillFailed=2
		ResultConnectFailed=255
	*/

	jobnum := 1
	success := 0
	fail := 0
	overtime := 0
	connect := 0
	for result := range results {
		switch result.resultcode {
		case ResultSuccess:
			if pdo.OutputWay != "row" {
				fmt.Printf("[%d/%d] %s \033[34m [SUCCESS]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			}
			CreateAppendFile(pdo.SuccessFile, result.jobname)
			success++
		case ResultTimeOverKilled:
			fmt.Printf("[%d/%d] %s \033[1;31m [Time Over KILLED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			CreateAppendFile(pdo.FailFile, result.jobname)
			overtime++
		case ResultKillFailed:
			fmt.Printf("[%d/%d] %s \033[1;31m [KILLED FAILED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			CreateAppendFile(pdo.FailFile, result.jobname)
			overtime++
		case ResultConnectFailed:
			fmt.Printf("[%d/%d] %s \033[1;31m [Connect FAILED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			CreateAppendFile(pdo.FailFile, result.jobname)
			connect++
		default:
			if pdo.OutputWay != "row" {
				fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", jobnum, pdo.JobTotal, result.jobname)
			}
			CreateAppendFile(pdo.FailFile, result.jobname)
			fail++
		}
		fmt.Println(result.resultinfo)
		jobnum++
	}
	fmt.Printf("[INFO] Total: %d ; Success: %d ; Failed: %d ; OverTime: %d ; ConnectFailed: %d\n", pdo.JobTotal, success, fail, overtime, connect)
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

/*
//创建bns service input来源
func CreateBnsServiceHost(bns string) (lists []HostList, err error) {

	bnsName := strings.Split(bns, ",")

	var list HostList
	//var lists []HostList

	for _, v := range bnsName {
		cmd := exec.Command("get_instance_by_service", "-a", v)
		//命令执行
		stdout, err := cmd.StdoutPipe()
		checkErr(2, err)
		stderr, err := cmd.StderrPipe()
		checkErr(2, err)
		err = cmd.Start()
		checkErr(2, err)

		go io.Copy(os.Stderr, stderr)

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			words := strings.Fields(scanner.Text())
			if len(words) >= 1 {
				list.Host = words[1]
				list.Path = HOME
				if FilterHosts(list.Host) && UniqHosts(lists, list) {
					lists = append(lists, list)
				}
			}
		}
		err = scanner.Err()
		checkErr(2, err)
	}
	return lists, err
}

//创建bns service input来源
func CreateNoahTreeHost(noah string) (lists []HostList, err error) {

	noahPath := strings.Split(noah, ",")

	var list HostList
	//var lists []HostList

	for _, v := range noahPath {
		cmd := exec.Command("get_hosts_by_path", v)
		//命令执行
		stdout, err := cmd.StdoutPipe()
		checkErr(2, err)
		stderr, err := cmd.StderrPipe()
		checkErr(2, err)
		err = cmd.Start()
		checkErr(2, err)

		go io.Copy(os.Stderr, stderr)

		scanner := bufio.NewScanner(stdout)
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
		err = scanner.Err()
		checkErr(2, err)
	}
	return lists, err
}
*/

//创建host列表  Input来源 文件或者管道
func CreateHostList(file io.Reader) ([]HostList, error) {
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

//文件追加内容
func CreateAppendFile(appendfile string, contents string) error {
	file, err := os.OpenFile(appendfile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
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
				CreateAppendFile(pdo.FailFile, pdo.JobList[x].Host+" "+pdo.JobList[x].Path)
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

	switch pdo.SubCommand {
	case "copy":
		//copy文件
		cmdLine := strings.Split(pdo.CmdLastString, "<:::>")
		ch := cmdLine[1][0]
		if string(ch) == "/" {
			cmd = exec.Command("rsync", "-e", "ssh -q -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", cmdLine[0], pdo.User+"@"+job.jobname.Host+":"+cmdLine[1])
		} else {
			cmd = exec.Command("rsync", "-e", "ssh -q -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", cmdLine[0], pdo.User+"@"+job.jobname.Host+":"+job.jobname.Path+"/"+cmdLine[1])
		}
	case "script":
		//脚本执行 先copy 后 执行
		cmdLine := strings.Split(pdo.CmdLastString, "<:::>")

		tmpl, err := template.New("pdo").Parse(cmdLine[2])
		checkErr(2, err)
		hostTemp := HostList{job.jobname.Host, job.jobname.Path}
		var tempBuf bytes.Buffer
		err = tmpl.Execute(&tempBuf, hostTemp)
		checkErr(2, err)
		argStr := tempBuf.String()

		remoteCmd := fmt.Sprintf("chmod +x %s && %s %s || cd /tmp/ && rm -f %s", cmdLine[1], cmdLine[1], argStr, cmdLine[1])
		copycmd := exec.Command("rsync", "-e", "ssh -q -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", cmdLine[0], pdo.User+"@"+job.jobname.Host+":"+cmdLine[1])
		copycmd.Run()
		cmd = exec.Command("ssh", "-q", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", pdo.User+"@"+job.jobname.Host, "cd", job.jobname.Path, "&&", remoteCmd)
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
			cmd = exec.Command("ssh", "-q", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", pdo.User+"@"+job.jobname.Host, "cd", job.jobname.Path, "&&", pdo.CmdLastString)
		}
	}

	//命令执行
	stdout, err := cmd.StdoutPipe()
	checkErr(2, err)
	stderr, err := cmd.StderrPipe()
	checkErr(2, err)
	err = cmd.Start()
	checkErr(2, err)
	//pid:=cmd.Process.Pid
	//fmt.Println("get pid ",pid)
	//执行过程内容
	//输出存储到文件
	if pdo.OutputDir != "" {
		//写重复host时候 并发有时间的先后顺序问题. 所以暂时使用array jobid进行记录.保证不会冲突
		outFile := fmt.Sprintf("%s/%s", pdo.OutputDir, job.jobname.Host)
		writeFile := outFile + "_" + strconv.Itoa(job.jobid)
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
				if pdo.Formula != "" {
					pdo.funcFormula(scanner.Text(), job.jobname.Host, job.jobname.Path)
				}
			}
		} else {
			for scanner.Scan() {
				fmt.Printf(">> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, scanner.Text())
				if pdo.Formula != "" {
					pdo.funcFormula(scanner.Text(), job.jobname.Host, job.jobname.Path)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			checkErr(2, err)
		}
		scannerErr := bufio.NewScanner(stderr)
		for scannerErr.Scan() {
			fmt.Printf(">> \033[34m%-25s\033[0m:FAIL>> %s\n", job.jobname.Host, scannerErr.Text())
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
			job.results <- Result{jobstring, ResultKillFailed, "Killed..."}
			checkErr(2, err)
		}
		<-done
		job.results <- Result{jobstring, ResultTimeOverKilled, "Time over ,Killed..."}
		//记录失败job
	case err := <-done:
		if err != nil {
			//完成返回失败
			var exitCode int

			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			}
			job.results <- Result{jobstring, exitCode, outerr.String()}
		} else {
			//完成返回成功
			if pdo.OutputWay != "row" {
				//如果是行显示就隐藏
				job.results <- Result{jobstring, ResultSuccess, out.String()}
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
		//fmt.Println(conffile + " not exist.")
		openconffile, err := os.OpenFile(conffile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		defer openconffile.Close()
		conftmpl, err := template.New("pdoconf").Parse(pdoconf)
		checkErr(1, err)
		err = conftmpl.Execute(openconffile, HOME)
		checkErr(1, err)

	}
	if !checkExist(logfile) {
		//fmt.Println(logfile + " not exist.")
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

/*
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

	//先拿到prev的列表
	preApp, _ := ListAppMysql(conn, proName+"-prev")
	for _, vpre := range preApp {
		Lists = append(Lists, vpre)
	}

	rows, err := stmtProductId.Query(proId)
	for rows.Next() {
		err = rows.Scan(&app, &appId)
		checkErr(2, err)
		if app != proName+"-prev" {
			arrApp, _ := ListAppMysql(conn, app)
			for _, va := range arrApp {
				Lists = append(Lists, va)
			}
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
			if *grephost != "" {
				if *grephost != hostname {
					continue
				}
			}

			L.Path = "/home/matrix/containers/" + containerId + "/home/work/orp"

			L.Host = hostname

			if FilterHosts(L.Host) && UniqHosts(Lists, L) {
				Lists = append(Lists, L)
			}
		}
	}
	return Lists, err
}
*/

//Formula 计算
func (pdo *Pdo) funcFormula(content string, host string, path string) {
	mula := strings.SplitN(pdo.Formula, ":", 3)
	column, err := strconv.Atoi(mula[0])
	checkErr(1, err)
	lenmula := len(mula)
	row := strings.Fields(content)
	lenrow := len(row)

	formula := make(map[string]string, 2)
	switch mula[1] {
	case "diff":
		if lenmula == 2 && column <= lenrow && column > 0 {
			recordFile := pdo.FormulaDir + "/" + row[0]
			formula[row[0]] = recordFile
			pdo.FormulaArray = formula
			CreateAppendFile(recordFile, row[0]+" "+host+" "+path)
		}
	case "add":
	case "gt":
	case "eq":
	default:
	}

}
