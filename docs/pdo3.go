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

const AppVersion = "Version 1.4.1"

var (
    SUDO_USER = os.Getenv("SUDO_USER")
    HOME      = os.Getenv("HOME")
    PID       = strconv.Itoa(os.Getppid())
    concurrent = 2 
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

type Result struct {
    jobname    string
    resultcode int
    resultinfo string
}

type Job struct {
    jobname HostList
    results chan<- Result
}

type PdoJobs struct {
    pdoinfo *PdoInfo
     
}

type HostList struct {
    Host string
    Path string
}

type TempScript struct {
    CMD string
}

type CmdString struct {
    HOST    string
}

type PdoInfo struct{
    Concurrent      int
    JobsList        []HostList
    JobsTotal       int
    JobsFinished    int
    CmdLastString   string
    PdoLocation     bool
    FailFile        string
    MatchString     string
    QuietOption     bool
    ShowOption      string
    OutputWay       string
    TimeWait        time.Duration
    TimeInterval    time.Duration
    SubCommand      string
    OutputDir       string
}

type Other struct {
    OutputDir       string
    CopyFile        string
    RetryOption     bool
    ScriptFile      string
    BuildInScript   string
    TemplateFile    string
    
}


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

func Filter(hostName string) bool {

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

//create List 
func CreateListFile(file *os.File) ([]HostList, error) {
    var L HostList
    var Lists []HostList

    scanner := bufio.NewScanner(file)

    for scanner.Scan() {
        words := strings.Fields(scanner.Text())
        if len(words) >= 1 {
            if len(words) < 2 {
                L.Host = words[0]
                L.Path = HOME
            } else {
                L.Host = words[0]
                L.Path = words[1]
            }
            if Filter(L.Host) {
                Lists = append(Lists, L)
            }
        }
    }
    err := scanner.Err()
    checkErr(2, err)
    return Lists, err
}

//catch system signal
func (Info *PdoInfo)sysSignalHandle() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        for sig := range c {
            log.Warnf("Ctrl+c,recode fail list to "+Info.FailFile+" ,signal:%s", sig)
            for x := Info.JobsFinished - 1; x <Info.JobsTotal ; x++ {
                CreateFailList(Info.FailFile, Info.JobsList[x].Host+" "+Info.JobsList[x].Path)
            }
            os.Exit(0)
        }
    }()
}

//create faile list
func CreateFailList(failFile string, contents string) error {
    file, err := os.OpenFile(failFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    defer file.Close()
    checkErr(2, err)
    file.WriteString(contents + "\n")
    return err
}

//func f(x int, wg *sync.WaitGroup, host string, dir string, cmdLine string, total int, output string) {
//
//func (job Job) Do() {
//
//    fmt.Printf("... doing work in [%s]\n",job.jobname.Host)
//    time.Sleep(time.Duration(rand.Float32() * float32(10* time.Second)))
//    t := time.Now()
//    fmt.Println(t.Format("20060102150405"))
//
//    if job.jobname.Host != "golang" {
//        job.results <- Result{job.jobname.Host, 0,"OK"}
//    } else {
//        job.results <- Result{job.jobname.Host,1,"Error"}
//    }
//}

func (job Job) Do(){

    var out, outerr bytes.Buffer
//
//    // whatever processing

    var tempBuf bytes.Buffer
    var cmd *exec.Cmd
    if info.PdoLocation {
        tmpl, err := template.New("pdo").Parse(info.CmdLastString)
        checkErr(2,err)
        err = tmpl.Execute(&tempBuf, job.jobname.Host) 
        checkErr(2,err)
        info.CmdLastString=tempBuf.String()
        cmd = exec.Command("/bin/sh","-c",info.CmdLastString)
    }else{
        cmd = exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", job.jobname.Host, "cd", job.jobname.Path, "&&", info.CmdLastString)
}
//    if *script != "" {
//        remoteCmd := fmt.Sprintf("chmod +x %s && %s && cd /tmp/ && rm -f %s", cmdLine, cmdLine, cmdLine)
//        copycmd := exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", *script, host+":"+cmdLine)
//        if err := copycmd.Run(); err != nil {
//            checkErr(2, err)
//            fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", info.JobsFinished, total, host)
//            CreateFailList(FailFile, host+" "+dir)
//            info.JobsFinished++
//            return
//        }
//
//        cmd = exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", host, "cd", dir, "&&", remoteCmd)
//
//    } else if *copy != "" {
//        ch := cmdLine[0]
//        if string(ch) == "/" {
//            cmd = exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", *copy, host+":"+cmdLine)
//        } else {
//            cmd = exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", *copy, host+":"+dir+"/"+cmdLine)
//        }
//    }
//
//    // Create stdout, stderr streams of type io.Reader
    stdout, err := cmd.StdoutPipe()
    checkErr(2, err)
    stderr, err := cmd.StderrPipe()
    checkErr(2, err)
//
//    // Start command
     err = cmd.Start()
     checkErr(2, err)
//
    if info.OutputDir != "" {
        outFile := fmt.Sprintf("%s/%s", info.OutputDir, job.jobname.Host)
        outf, err := os.Create(outFile)
        defer outf.Close()
        checkErr(2, err)
        go io.Copy(outf, stdout)
        go io.Copy(outf, stderr)
    } else if info.OutputWay == "row" {
        scanner := bufio.NewScanner(stdout)
        if info.MatchString != "" {
            for scanner.Scan() {
                if strings.Contains(scanner.Text(), info.MatchString) {
                    fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", job.jobname.Host, strings.Replace(scanner.Text(), info.MatchString, "\033[1;31m"+info.MatchString+"\033[0m", -1))
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
    case <-time.After(info.TimeWait):
        if err := cmd.Process.Kill(); err != nil {
            fmt.Printf("[%d/%d] %s \033[1;31m [KILL FAILED]\033[0m.\n", info.JobsFinished, info.JobsTotal, job.jobname.Host)
            checkErr(2, err)
        }
        <-done // allow goroutine to exit
        fmt.Printf("[%d/%d] %s \033[1;31m [Time Over KILLED]\033[0m.\n",  info.JobsFinished, info.JobsTotal, job.jobname.Host)
        CreateFailList(info.FailFile, job.jobname.Host+" "+job.jobname.Path)
    case err := <-done:
        if err != nil {
            fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", info.JobsFinished, info.JobsTotal, job.jobname.Host)
            CreateFailList(info.FailFile, job.jobname.Host+" "+job.jobname.Path)
            if info.OutputDir == "" && info.OutputWay == "" {
                fmt.Println(outerr.String())
            }
       } else {
            if info.OutputWay == "" {
               fmt.Printf("[%d/%d] %s \033[34m [SUCCESS]\033[0m.\n", info.JobsFinished, info.JobsTotal, job.jobname.Host)
                //get the signal about the finished hosts.
                if info.OutputDir == "" {
                    fmt.Println(out.String())
                }
            }
        }
    }
//
    info.JobsFinished++
}

//func (Info *PdoInfo)ParseTemplate() bool {
//
//    var text string
//    var doc bytes.Buffer
//
//    if Info.BuildInScript != "" {
//        inScript, _ := ioutil.ReadFile(Info.BuildInScript)
//        text = string(inScript)
//    } else {
//        text = Info.CmdLastString
//    }
//    fin, err := ioutil.ReadFile(Info.TemplateFile)
//    if err != nil {
//        fmt.Println(err)
//    }
//
//    Temp := TempScript{
//        CMD: text,
//    }
//    //output file
//    outf, err := os.Create(Info.OutputDir)
//    defer outf.Close()
//    if err != nil {
//        fmt.Println(err)
//        return false
//    }
//
//    t := template.New("script template")
//    t, err = t.Parse(string(fin))
//    checkErr(2, err)
//    err = t.Execute(&doc, Temp)
//    checkErr(2, err)
//    io.Copy(outf, &doc)
//
//    return true
//}
//

func NewJob(info *PdoInfo) *Job{
 job := &Job{pdo: info}
 return job
}

func doRequest(job *Job) {

    jobs := make(chan Job, job.pdoinfo.Concurrent)
    results := make(chan Result, len(job.jobname))
    done := make(chan struct{}, job.pdoinfo.Concurrent)

    go addJobs(jobs, jobnames, results,job.pdoinfo)

    for i := 0; i < job.pdoinfo.Concurrent; i++ {
        go doJobs(done, jobs,) 
    }

    go awaitCompletion(done, results)

    processResults(results)
}

func addJobs(jobs chan<- Job, jobnames []HostList, results chan<- Result,info *PdoInfo) {
    for _, jobname := range jobnames {
        jobs <- Job{jobname,results,info}
    }
    close(jobs)
}

func doJobs(done chan<- struct{}, jobs <-chan Job) {

    for job := range jobs {
       job.Do(job)
    }
    done <- struct{}{}
}

func awaitCompletion(done <-chan struct{}, results chan Result) {
    for i := 0; i < *workers; i++ {
        <-done
    }
    close(results)
}

func processResults(results <-chan Result) {
    for result := range results {
        fmt.Printf("done: %s,%d,%s\n", result.jobname, result.resultcode, result.resultinfo)
    }
}



func main() {
    var tempCommand string
    runtime.GOMAXPROCS(runtime.NumCPU())

    //ini configure
    conf, err := config.ReadDefault(*configFile)
    checkErr(1, err)

    //log record
    logConf, _ := conf.String("PDO", "logconf")
    logger, _ := log.LoggerFromConfigAsFile(logConf)
    log.ReplaceLogger(logger)

    //record input args
    log.Info(SUDO_USER + " " + PID + " Do: " + strings.Join(os.Args, " "))
    log.Flush()


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
    if strings.Contains(tempCommand, "{{.HOST}}") { 
        templateTrue=true
    }

    Info:=&PdoInfo{
        Concurrent    : *workers ,
        OutputDir     : *outputDir ,
        FailFile      : "/tmp/pdo_faile." + PID ,
        MatchString   : *mstring ,
        ShowOption    : *outputShow ,
        QuietOption   : *quiet ,
        PdoLocation   :  templateTrue,
        TimeWait      : *waitTime,
        TimeInterval  : *interTime,
        CmdLastString : tempCommand ,
    }

    if *retry {
        sourceFile, err := os.Open(Info.FailFile)
        checkErr(1, err)
        Info.JobsList, _ = CreateListFile(sourceFile)
    } else if *hostFile != "" {
        sourceFile, err := os.Open(*hostFile)
        checkErr(1, err)
        Info.JobsList, _ = CreateListFile(sourceFile)
    } else {
        Info.JobsList, _ = CreateListFile(os.Stdin)
    }

    Info.JobsTotal=len(Info.JobsList)

    if *shortCmd!= "" {
        conf, _ := config.ReadDefault(*configFile)
        Info.CmdLastString, _ = conf.String("CMD", *shortCmd)
    }

    log.Info(Info.JobsList)
    defer log.Flush()

    //使用template
    if *scriptTemp != "" {
        conf, _ := config.ReadDefault(*configFile)
        Info.TemplateFile, _ = conf.String("TEMPLATE", *scriptTemp)
        Info.ScriptFile="/tmp/pdo_script."+PID
        if Info.ParseTemplate() {
            Info.CmdLastString= "/tmp/pdo_script." + PID
        }
    }

    //执行脚本
    if Info.ScriptFile != "" {
        Info.CmdLastString = "/tmp/pdo_script." + PID
    }

    //没有输入命令
    if Info.CmdLastString == "" {
        fmt.Println("[ Error ] cmd input error ,no command or action need to do ! exit...")
        os.Exit(1)
    }

    //头部输出
    if !Info.QuietOption {
        fmt.Println(">>>> Welcome " + SUDO_USER + "...")
        for i, elem := range Info.JobsList {
            fmt.Printf("%-25s-%-20s ", elem.Host, elem.Path)
            if (i+1)%2 == 0 {
                fmt.Printf("\n")
            }
        }
        fmt.Printf("\n")
        fmt.Println("#--Total--# ", Info.JobsTotal)
        if Info.CopyFile != "" {
            fmt.Println("#---CMD---# ", *copy, "-->", Info.CmdLastString)
        } else if Info.ScriptFile != "" {
            fmt.Println("#---CMD---# ", "Script:", Info.ScriptFile)
        } else {
            fmt.Println("#---CMD---# ", Info.CmdLastString)
        }
    }

    if Info.JobsTotal == 0 {
        fmt.Println("No Host List !Please check input . exit...")
        os.Exit(1)
    }

    YesNO()

    //create fail list
    os.Create(Info.FailFile)

    //mkdir  output
    if Info.OutputDir != "" {
        err := os.MkdirAll(Info.OutputDir, 0755)
        checkErr(2, err)
    }

    job := NewJob(&Info)
    doRequest(job)

//    //concurent control
//    var wg sync.WaitGroup
//    var num, max int
//    realMax := len(Arrhost)
//    pdoMax := realMax - 1
//
//    var N = concurrent
//
//
//    //do the first
//    if !*yesorno {
//        wg.Add(1)
//        f(0, &wg, Arrhost[0].Host, Arrhost[0].Path, cmdLine, len(Arrhost), *outputDir)
//        YesNO()
//    } else {
//        wg.Add(1)
//        go f(0, &wg, Arrhost[0].Host, Arrhost[0].Path, cmdLine, len(Arrhost), *outputDir)
//    }
//
//    max = pdoMax / N
//    if pdoMax%N != 0 {
//        max += 1
//    }
//
//    log.Flush()
//
//    //get signal , do something.
//    go sysSignalHandle()
//
//    for x := 0; x < max; x++ {
//        num = N
//        if pdoMax <= N {
//            num = pdoMax
//        }
//        for i := 0; i < num; i++ {
//            //throttle <- 1
//            wg.Add(1)
//            hostNow := x*N + i + 1
//            go f(x, &wg, Arrhost[hostNow].Host, Arrhost[hostNow].Path, cmdLine, len(Arrhost), *outputDir)
//        }
//        pdoMax -= num
//        wg.Wait()
//        time.Sleep(*interTime)
//    }
//
}
