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
    "time"
    "runtime"
    log "github.com/cihub/seelog"
    "github.com/robfig/config"
	)

const AppVersion = "Version 2.0.1"

var (
    SUDO_USER = os.Getenv("SUDO_USER")
    HOME      = os.Getenv("HOME")
    PID       = strconv.Itoa(os.Getppid())
    workers    = flag.Int("r", 1, "concurrent processing ,default 1 .")
    waitTime   = flag.Duration("t", 300*time.Second, "reach the time ,kill process.")
    interTime  = flag.Duration("T", 0*time.Second, "Interval between concurrent programs.")
    hostFile   = flag.String("f", "", "host list,allow 1 or 2 columns. the first column is HostName or IP , the second column is path.")
    outputDir  = flag.String("o", "", "output dir. ")
    outputShow = flag.String("show", "", "show option,you can use <row>")
    product    = flag.String("p", "", "input product name.")
    app        = flag.String("a", "", "input app name.")
    mstring    = flag.String("match", "", "match a string with color.")
    mrule      = flag.String("rule", "", "rule for match string in conf.")
    copy       = flag.String("c", "", "Copy <file> to destination as <dest_path>. If notspecified, <dest_path> will be same as <file>.")
    script     = flag.String("e", "", "Transfer/execute a script from the local system to eachtarget system.")
    idc        = flag.String("i", "", "input filter idc name.")
    idcs       = flag.String("I", "", "filter JX or TC .")
    shortCmd   = flag.String("cmd", "", "short command in conf.")
    configFile = flag.String("C", HOME+"/.pdo/pdo.conf", "configure file , default ~/.pdo/pdo.conf.")
    yesorno    = flag.Bool("y", false, "continue, yes/no , default true.")
    retry      = flag.Bool("R", false, "retry fail list at the last time")
    quiet      = flag.Bool("q", false, "quiet mode,disable header output.")
    scriptTemp = flag.String("temp", "", "use template")
    builtin    = flag.String("b", "", "built-in for template,when use temp")
    version    = flag.Bool("V", false, "Print the app version.")
)


type Pdo struct{
    Concurrent      int
    JobList        []HostList
    JobTotal       int
    JobFinished    int
    CmdLastString   string
    PdoLocal        bool
    FailFile        string
    MatchString     string
    QuietOption     bool
    ShowOption      string
    OutputWay       string
    TimeWait        time.Duration
    TimeInterval    time.Duration
    SubCommand      string
    OutputDir       string
    TemplateFile    string
    ScriptFile      string
    CopyFile        string
    Jobs             Job
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
	pdo:=NewPdo()

	//头部显示 
	pdo.displayHead()

    YesNO()

	//并发调度处理
	pdo.doRequest()

	//尾部显示
	pdo.displayEnd()
}

//#################
//主体main的相关函数 
//获取和判断相关信息
func NewPdo() *Pdo {

    var tempCommand string
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


    //参数解析 
    flag.Parse()

    if *version {
        fmt.Println(AppVersion)
        os.Exit(0)
    }

    preCommand:=flag.Args()[0]
    switch preCommand {
    case  "config" : 
        tempCommand=flag.Args()[1]
        
    default: 

        tempCommand=flag.Args()[0]

    }

    templateTrue:=false
    if strings.Contains(tempCommand, "{{.}}") || strings.Contains(tempCommand,"{{.Host}}") || strings.Contains(tempCommand,"{{.Path}}") { 
        templateTrue=true
    }

    info:=&Pdo{
        Concurrent    : *workers ,
        OutputDir     : *outputDir ,
        FailFile      : "/tmp/pdo_faile." + PID ,
        MatchString   : *mstring ,
        ShowOption    : *outputShow ,
        QuietOption   : *quiet ,
        PdoLocal      : templateTrue,
        TimeWait      : *waitTime,
        TimeInterval  : *interTime,
        CmdLastString : tempCommand ,
    }

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

    info.JobTotal=len(info.JobList)

    if *shortCmd!= "" {
        conf, _ := config.ReadDefault(*configFile)
        info.CmdLastString, _ = conf.String("CMD", *shortCmd)
    }

    log.Info(info.JobList)
    defer log.Flush()

    //执行脚本
    if info.ScriptFile != "" {
        info.CmdLastString = "/tmp/pdo_script." + PID
    }

    //没有输入命令
    if info.CmdLastString == "" {
        fmt.Println("[ Error ] cmd input error ,no command or action need to do ! exit...")
        os.Exit(1)
    }

    return info

}

//头部信息显示显示 
func (pdo *Pdo)displayHead() {
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
        if pdo.CopyFile != "" {
            fmt.Println("#---CMD---# ", *copy, "-->", pdo.CmdLastString)
        } else if pdo.ScriptFile != "" {
            fmt.Println("#---CMD---# ", "Script:", pdo.ScriptFile)
        } else {
            fmt.Println("#---CMD---# ", pdo.CmdLastString)
        }
    }

    if pdo.JobTotal == 0 {
        fmt.Println("No Host List !Please check input . exit...")
        os.Exit(1)
    }

}

//尾部信息显示 
func (pdo *Pdo)displayEnd() {
	
}


//####################
//并发调度过程
//处理job对列 
//并发调度开始
func (pdo *Pdo)doRequest() {
	jobs := make(chan Job, pdo.Concurrent)
    results := make(chan Result, len(pdo.JobList))
    done := make(chan struct{}, pdo.Concurrent)

    go pdo.addJob(jobs, pdo.JobList, results)


    for i := 0; i < pdo.Concurrent; i++ {
        go pdo.doJob(done, jobs) 
    }

    go awaitCompletion(done, results,pdo.Concurrent)

    processResults(results)
}

//添加job
func (pdo *Pdo)addJob(jobs chan<- Job, jobnames []HostList, results chan<- Result) {
    for num, jobname := range jobnames {
        jobs <- Job{jobname,results}
        if ! pdo.QuietOption && num == 0{
           for {
                if pdo.JobFinished == 1 {
                    YesNO() 
                    break
                }else{
                    time.Sleep(1 * time.Second) 
                }
            } 
        }
    }
    close(jobs)
}

//处理job
func (pdo *Pdo)doJob(done chan<- struct{}, jobs <-chan Job) {

    for job := range jobs {
       pdo.Do(&job)
    }
    done <- struct{}{}
}

//job完成状态
func awaitCompletion(done <-chan struct{}, results chan Result,works int) {
    for i := 0; i < works; i++ {
        <-done
    }
    close(results)
}

//job处理结果
func processResults(results <-chan Result) {
    for result := range results {
        fmt.Printf("done: %s,%d,%s\n", result.jobname, result.resultcode, result.resultinfo)
    }
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

//判断Yes or No的输入
func YesNO() {
    userFile := "/dev/tty"
    if !*yesorno {
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
            if FilterHosts(list.Host) {
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
func (pdo *Pdo)sysSignalHandle() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        for sig := range c {
            log.Warnf("Ctrl+c,recode fail list to "+pdo.FailFile+" ,signal:%s", sig)
            for x := pdo.JobFinished - 1; x <pdo.JobTotal ; x++ {
                CreateFailFile(pdo.FailFile, pdo.JobList[x].Host+" "+pdo.JobList[x].Path)
            }
            os.Exit(0)
        }
    }()
}


//具体job过程
func (pdo *Pdo) Do(job *Job){

    pdo.JobFinished++
    var out, outerr bytes.Buffer
    var cmd *exec.Cmd
    if pdo.PdoLocal {
        tmpl, err := template.New("pdo").Parse(pdo.CmdLastString)
        checkErr(2,err)
        hostTemp:=HostList{job.jobname.Host,job.jobname.Path}
        var tempBuf bytes.Buffer
        err = tmpl.Execute(&tempBuf,hostTemp) 
        checkErr(2,err)
        outputString:=tempBuf.String()
        fmt.Println(outputString)
        cmd = exec.Command("/bin/bash", "-s")
        cmd.Stdin = bytes.NewBufferString(outputString)
    }else{
        cmd = exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", job.jobname.Host, "cd", job.jobname.Path, "&&", pdo.CmdLastString)
}

    stdout, err := cmd.StdoutPipe()
    checkErr(2, err)
    stderr, err := cmd.StderrPipe()
    checkErr(2, err)
//
//    // Start command
     err = cmd.Start()
     checkErr(2, err)
//
    if pdo.OutputDir != "" {
        outFile := fmt.Sprintf("%s/%s", pdo.OutputDir, job.jobname.Host)
        outf, err := os.Create(outFile)
        defer outf.Close()
        checkErr(2, err)
        go io.Copy(outf, stdout)
        go io.Copy(outf, stderr)
    } else if pdo.OutputWay == "row" {
        scanner := bufio.NewScanner(stdout)
        if pdo.MatchString != "" {
            for scanner.Scan() {
                if strings.Contains(scanner.Text(), pdo.MatchString) {
                    fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, strings.Replace(scanner.Text(), pdo.MatchString, "\033[1;31m"+pdo.MatchString+"\033[0m", -1))
                } else {
                    fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, scanner.Text())
                }
            }
        } else {
            for scanner.Scan() {
                fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, scanner.Text())
            }
        }
        if err := scanner.Err(); err != nil {
            checkErr(2, err)
        }

    } else {
        go io.Copy(&out, stdout)
        go io.Copy(&outerr, stderr)
    }
//
    done := make(chan error)

    go func() {
        done <- cmd.Wait()
    }()

    select {
    case <-time.After(pdo.TimeWait):
        if err := cmd.Process.Kill(); err != nil {
            fmt.Printf("[%d/%d] %s \033[1;31m [KILL FAILED]\033[0m.\n", pdo.JobFinished, pdo.JobTotal, job.jobname.Host)
            checkErr(2, err)
        }
        <-done // allow goroutine to exit
        fmt.Printf("[%d/%d] %s \033[1;31m [Time Over KILLED]\033[0m.\n",  pdo.JobFinished, pdo.JobTotal, job.jobname.Host)
        CreateFailFile(pdo.FailFile, job.jobname.Host+" "+job.jobname.Path)
    case err := <-done:
        if err != nil {
            fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", pdo.JobFinished, pdo.JobTotal, job.jobname.Host)
            CreateFailFile(pdo.FailFile, job.jobname.Host+" "+job.jobname.Path)
            if pdo.OutputDir == "" && pdo.OutputWay == "" {
                fmt.Println(outerr.String())
            }
       } else {
            if pdo.OutputWay == "" {
               fmt.Printf("[%d/%d] %s \033[34m [SUCCESS]\033[0m.\n", pdo.JobFinished, pdo.JobTotal, job.jobname.Host)
                //get the signal about the finished hosts.
                if pdo.OutputDir == "" {
                    fmt.Println(out.String())
                }
            }
        }
    }
//
}




