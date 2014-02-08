package main

import (
    "bufio"
    "bytes"
    "database/sql"
    "flag"
    "fmt"
    "io"
    "io/ioutil"
    "os"
    "os/exec"
    "os/signal"
    "strconv"
    "strings"
    "sync"
    "text/template"
    "time"
    log "github.com/cihub/seelog"
    "github.com/robfig/config"
)

const maxConcurrency = 4 // for example
const AppVersion = "Version 1.2.2"

var (
    SUDO_USER = os.Getenv("SUDO_USER")
    HOME      = os.Getenv("HOME")

    concurrent = flag.Int("r", 1, "concurrent processing ,default 1 .")
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
    cmds       = flag.String("cmd", "", "short command in conf.")
    configFile = flag.String("C", HOME+"/.pdo/pdo.conf", "configure file , default ~/.pdo/pdo.conf.")
    yesorno    = flag.Bool("y", false, "continue, yes/no , default true.")
    retry      = flag.Bool("R", false, "retry fail list at the last time")
    quiet      = flag.Bool("q", false, "quiet mode,disable header output.")
    scriptTemp = flag.String("temp", "", "use template")
    builtin    = flag.String("b", "", "built-in for template,when use temp")
    version    = flag.Bool("V", false, "Print the app version.")

    HostNum  = 1
    FailFile string
    Arrhost  []HostList
)

type HostList struct {
    Host string
    Path string
}

type TempScript struct {
    CMD string
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

//parse file , .
func ListFile(file *os.File) ([]HostList, error) {
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

func main() {
    flag.Parse()
    if *version {
        fmt.Println(AppVersion)
        os.Exit(0)
    }

    var conn, sTemplate string
    cmdline := strings.Join(flag.Args(), "\"")
    //pid, _ := GetPID()
    pid := strconv.Itoa(os.Getppid())
    FailFile = "/tmp/pdo/" + pid

    //ini configure
    c, err := config.ReadDefault(*configFile)
    checkErr(1, err)

    //log record
    logConf, _ := c.String("PDO", "logconf")

    logger, _ := log.LoggerFromConfigAsFile(logConf)
    log.ReplaceLogger(logger)

    //record input args
    log.Info(SUDO_USER + " " + pid + " Do: " + strings.Join(os.Args, " "))
    log.Flush()
    if *product != "" || *app != "" {
        dbhost, err := c.String("Mysql", "Host")
        dbport, err := c.String("Mysql", "Port")
        dbuser, err := c.String("Mysql", "User")
        dbpass, err := c.String("Mysql", "Pass")
        dbname, err := c.String("Mysql", "DBname")
        conn = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", dbuser, dbpass, dbhost, dbport, dbname)
        checkErr(1, err)
    }

    if *hostFile != "" {
        file, err := os.Open(*hostFile)
        checkErr(1, err)
        Arrhost, _ = ListFile(file)
    } else if *retry {
        file, err := os.Open(FailFile)
        checkErr(1, err)
        Arrhost, _ = ListFile(file)
    } else {
        Arrhost, _ = ListFile(os.Stdin)
    }

    //concurent control

    var wg sync.WaitGroup
    var num, max int
    realMax := len(Arrhost)
    pdoMax := realMax - 1

    var N = *concurrent
    if *cmds != "" {
        c, _ := config.ReadDefault(*configFile)
        cmdline, _ = c.String("CMD", *cmds)
    }

    log.Info(Arrhost)

    defer log.Flush()

    //use template
    if *scriptTemp != "" {
        c, _ := config.ReadDefault(*configFile)
        sTemplate, _ = c.String("TEMPLATE", *scriptTemp)
        if ParseTemplate(sTemplate, "/tmp/pdo_script."+pid, cmdline) {
            cmdline = "/tmp/pdo_script." + pid
            *script = "/tmp/pdo_script." + pid
        }
    }

    //do script
    if *script != "" {
        cmdline = "/tmp/pdo_script." + pid
    }

    //print
    if !*quiet {
        fmt.Println(">>>> Welcome " + SUDO_USER + "...")
        for i, elem := range Arrhost {
            fmt.Printf("%-25s-%-20s ", elem.Host, elem.Path)
            if (i+1)%2 == 0 {
                fmt.Printf("\n")
            }
        }
        fmt.Printf("\n")
        fmt.Println("#--Total--# ", realMax)
        if *copy != "" {
            fmt.Println("#---CMD---# ", *copy, "-->", cmdline)
        } else if *script != "" {
            fmt.Println("#---CMD---# ", "Script:", *script)
        } else {
            fmt.Println("#---CMD---# ", cmdline)
        }
    }

    if cmdline == "" {
        fmt.Println("Please input command ! exit...")
        os.Exit(1)
    }

    if realMax == 0 {
        fmt.Println("No Host List ! exit...")
        os.Exit(1)
    }

    YesNO()

    //create fail list
    err = os.MkdirAll("/tmp/pdo", 0766)
    checkErr(2, err)
    os.Create(FailFile)

    //mkdir  output
    if *outputDir != "" {
        err := os.MkdirAll(*outputDir, 0755)
        checkErr(2, err)
    }

    //do the first
    if !*yesorno {
        wg.Add(1)
        f(0, &wg, Arrhost[0].Host, Arrhost[0].Path, cmdline, len(Arrhost), *outputDir)
        YesNO()
    } else {
        wg.Add(1)
        go f(0, &wg, Arrhost[0].Host, Arrhost[0].Path, cmdline, len(Arrhost), *outputDir)
    }

    max = pdoMax / N
    if pdoMax%N != 0 {
        max += 1
    }

    log.Flush()

    //get signal , do something.
    go sysSignalHandle()

    for x := 0; x < max; x++ {
        num = N
        if pdoMax <= N {
            num = pdoMax
        }
        for i := 0; i < num; i++ {
            //throttle <- 1
            wg.Add(1)
            hostNow := x*N + i + 1
            go f(x, &wg, Arrhost[hostNow].Host, Arrhost[hostNow].Path, cmdline, len(Arrhost), *outputDir)
        }
        pdoMax -= num
        wg.Wait()
        time.Sleep(*interTime)
    }

}

//do with signal
func sysSignalHandle() {
    c := make(chan os.Signal, 1)
    signal.Notify(c, os.Interrupt)
    go func() {
        for sig := range c {
            log.Warnf("Ctrl+c,recode fail list to "+FailFile+" ,signal:%s", sig)
            for x := HostNum - 1; x < len(Arrhost); x++ {
                FailList(FailFile, Arrhost[x].Host+" "+Arrhost[x].Path)
            }
            os.Exit(0)
        }
    }()
}

//create faile list
func FailList(failFile string, contents string) error {
    file, err := os.OpenFile(failFile, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    defer file.Close()
    checkErr(2, err)
    file.WriteString(contents + "\n")
    return err
}

//thread do
func f(x int, wg *sync.WaitGroup, host string, dir string, cmdline string, total int, output string) {

    var out, outerr bytes.Buffer
    defer wg.Done()

    // whatever processing
    cmd := exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", host, "cd", dir, "&&", cmdline)
    if *script != "" {
        remoteCmd := fmt.Sprintf("chmod +x %s && %s && cd /tmp/ && rm -f %s", cmdline, cmdline, cmdline)
        copycmd := exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", *script, host+":"+cmdline)
        if err := copycmd.Run(); err != nil {
            checkErr(2, err)
            fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", HostNum, total, host)
            FailList(FailFile, host+" "+dir)
            HostNum++
            return
        }

        cmd = exec.Command("ssh", "-xT", "-o", "PasswordAuthentication=no", "-o", "StrictHostKeyChecking=no", "-o", "ConnectTimeout=3", host, "cd", dir, "&&", remoteCmd)

    } else if *copy != "" {
        ch := cmdline[0]
        if string(ch) == "/" {
            cmd = exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", *copy, host+":"+cmdline)
        } else {
            cmd = exec.Command("rsync", "-e", "ssh -o PasswordAuthentication=no -o StrictHostKeyChecking=no -o ConnectTimeout=3", "-a", *copy, host+":"+dir+"/"+cmdline)
        }
    }

    // Create stdout, stderr streams of type io.Reader
    stdout, err := cmd.StdoutPipe()
    checkErr(2, err)
    stderr, err := cmd.StderrPipe()
    checkErr(2, err)

    // Start command
    err = cmd.Start()
    checkErr(2, err)

    // Non-blockingly echo command output to terminal
    if output != "" {
        outFile := fmt.Sprintf("%s/%s", output, host)
        outf, err := os.Create(outFile)
        defer outf.Close()
        checkErr(2, err)
        go io.Copy(outf, stdout)
        go io.Copy(outf, stderr)
    } else if *outputShow == "row" {
        scanner := bufio.NewScanner(stdout)
        if *mstring != "" {
            for scanner.Scan() {
                if strings.Contains(scanner.Text(), *mstring) {
                    fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", host, strings.Replace(scanner.Text(), *mstring, "\033[1;31m"+*mstring+"\033[0m", -1))
                } else {
                    fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", host, scanner.Text())
                }
            }
        } else {
            for scanner.Scan() {
                fmt.Printf("> \033[34m%-25s\033[0m >> %s\n", host, scanner.Text())
            }
        }
        if err := scanner.Err(); err != nil {
            checkErr(2, err)
        }

    } else {
        go io.Copy(&out, stdout)
        go io.Copy(&outerr, stderr)
    }

    done := make(chan error)

    go func() {
        done <- cmd.Wait()
    }()

    select {
    case <-time.After(*waitTime):
        if err := cmd.Process.Kill(); err != nil {
            fmt.Printf("[%d/%d] %s \033[1;31m [KILL FAILED]\033[0m.\n", HostNum, total, host)
            checkErr(2, err)
        }
        <-done // allow goroutine to exit
        fmt.Printf("[%d/%d] %s \033[1;31m [Time Over KILLED]\033[0m.\n", HostNum, total, host)
        FailList(FailFile, host+" "+dir)
    case err := <-done:
        if err != nil {
            fmt.Printf("[%d/%d] %s \033[1;31m [FAILED]\033[0m.\n", HostNum, total, host)
            FailList(FailFile, host+" "+dir)
            if output == "" && *outputShow == "" {
                fmt.Println(outerr.String())
            }
        } else {
            if *outputShow == "" {
                fmt.Printf("[%d/%d] %s \033[34m [SUCCESS]\033[0m.\n", HostNum, total, host)
                //get the signal about the finished hosts.
                if output == "" {
                    fmt.Println(out.String())
                }
            }
        }
    }

    HostNum++
}

func ParseTemplate(temp string, out string, cmdline string) bool {

    var text string
    var doc bytes.Buffer

    if *builtin != "" {
        inScript, _ := ioutil.ReadFile(*builtin)
        text = string(inScript)
    } else {
        text = cmdline
    }
    fin, err := ioutil.ReadFile(temp)
    if err != nil {
        fmt.Println(err)
    }

    Temp := TempScript{
        CMD: text,
    }
    //output file
    outf, err := os.Create(out)
    defer outf.Close()
    if err != nil {
        fmt.Println(err)
        return false
    }

    t := template.New("script template")
    t, err = t.Parse(string(fin))
    checkErr(2, err)
    err = t.Execute(&doc, Temp)
    checkErr(2, err)
    io.Copy(outf, &doc)

    return true
}
