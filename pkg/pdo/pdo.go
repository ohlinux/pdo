package pdo

import (
	"fmt"
    "os"
   "time"
   "sync"
   "os/signal"
   "log"
)


func NewPdo(pdoMaster Pdo)Pdo{

	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return pdoMaster

}

func  (pdo *Pdo)Run(){

	//todo: add debug log
//	fmt.Printf("run pdo %#v\n",pdo)
	// display header information
	if pdo.Output.Header {
		pdo.displayHeader()
	}

	if ! pdo.Parallel.Yes {
		YesNO()
	}

	// create success and fail file.
	if err:=pdo.PrepareWorkEnv() ; err!=nil {
		fmt.Println("parepare work env faild,",err)
		os.Exit(1)
	}

	// do job request
	pdo.DoRequest()

	// display summary information
	if pdo.Output.Summary {
		pdo.displaySummary()
	}
}


//####################
//并发调度过程
//处理job对列
//并发调度开始
func (pdo *Pdo) DoRequest() {

	jobs := make(chan Job, pdo.Parallel.Numbers)
	results := make(chan JobResult, pdo.Jobsinfo.Total)
	done := make(chan struct{}, pdo.Parallel.Numbers)

	go pdo.addJob(jobs, pdo.JobList, results)

	for i := 0; i < pdo.Parallel.Numbers; i++ {
		go pdo.doJob(done, jobs)
	}

	go pdo.sysSignalHandle()

	go pdo.awaitCompletion(done, results, pdo.Parallel.Numbers)

	pdo.processResults(results)
}

var waitgroup sync.WaitGroup

//添加job
func (pdo *Pdo) addJob(jobs chan<- Job, jobnames []HostList, results chan<- JobResult) {
	for num, jobname := range jobnames {

		waitgroup.Add(1)
		jobs <- Job{jobname, results, num + 1}
		// the first one
		if !pdo.Parallel.Yes {
			if num == 0 || (num%pdo.Parallel.Ask == 0 && num != (pdo.Jobsinfo.Total-1)) {
				waitgroup.Wait()
				YesNO()
			}
		}
	}
	close(jobs)
}

//处理job
func (pdo *Pdo) doJob(done chan<- struct{}, jobs <-chan Job) {

	for job := range jobs {
		pdo.Do(&job)
		time.Sleep(pdo.Parallel.IntervalTime)
	}
	done <- struct{}{}
}

//job完成状态
func (pdo *Pdo) awaitCompletion(done <-chan struct{}, results chan JobResult, works int) {
	for i := 0; i < works; i++ {
		<-done
	}
	close(results)
}

//job处理结果
func (pdo *Pdo) processResults(results <-chan JobResult) {
	jobfinish:=1

	var status string
	for result := range results {
		switch result.RetCode {
			case ResultSuccess:
				status=StatusSuccess
				pdo.Jobsinfo.Success++
			case ResultTimeOverKilled:
				status=StatusTimeOverKilled
				pdo.Jobsinfo.TimeOver++
			case ResultKillFailed:
				status=StatusTimeOverKilledFailed
				pdo.Jobsinfo.TimeOver++
			case ResultConnectFailed:
				status=StatusConnectFailed
				pdo.Jobsinfo.ConnectFail++
			default:
				status=StatusDoFailed
				pdo.Jobsinfo.ExeFail++
		}

		if status==StatusSuccess {
			//todo: add passwd ... and crypto
			pdo.CreateAppendFile(pdo.WorkEnv.SuccessFile, result.Jobname)
		}else{
			pdo.CreateAppendFile(pdo.WorkEnv.FailFile, result.Jobname)
		}

		//todo: if format is row , how to add  progress
		if pdo.Output.Format == FormatText || len(pdo.Output.Save) > 0{
			if status==StatusSuccess {
				fmt.Printf("[%d/%d] %s \033[34m [%s]\033[0m\n", jobfinish, pdo.Jobsinfo.Total, result.Jobname,status)
			}else{
				fmt.Printf("[%d/%d] %s \033[1;31m [%s]\033[0m\n", jobfinish, pdo.Jobsinfo.Total, result.Jobname,status)
			}
		}

		if pdo.Output.Format== FormatText && len(pdo.Output.Save) == 0 {
			fmt.Printf("%s\n",result.Stdout)
			fmt.Printf("%s\n",result.Stderr)
		}

		if pdo.Output.Format == FormatJson {
			pdo.OutputFormatJson(result)
		}

		if pdo.Output.Format == FormatYaml{
			pdo.OutputFormatYaml(result)
		}

		jobfinish++
		waitgroup.Done()
		pdo.Jobsinfo.Finished++
	}
}

//文件追加内容
func (pdo *Pdo)CreateAppendFile(appendfile *os.File, contents string) error {
	_,err:=appendfile.WriteString(contents + "\n")
	//todo: add debug info
	return err
}

//判断Yes or No的输入
func YesNO() {
	userFile := "/dev/tty"
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
			fmt.Println("next ...")
		case "n\n":
			fmt.Println("exit ...")
			os.Exit(1)
		default:
			goto here
		}
}

//信号处理
func (pdo *Pdo) sysSignalHandle() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			fmt.Printf("Ctrl+c,recode fail list to ,signal:%s\n", sig)
			for x := pdo.Jobsinfo.Finished - 1; x < pdo.Jobsinfo.Total; x++ {
				//todo: save format
				pdo.CreateAppendFile(pdo.WorkEnv.FailFile, pdo.JobList[x].Host+" "+pdo.JobList[x].Path)
			}
			os.Exit(0)
		}
	}()
}

//
//func syncLogger(logFile string, logLevel string, logFileMax int, logBackups int) {
//
//	var LogConfig = `
//<seelog minlevel="` + logLevel + `">
//    <outputs formatid="common">
//        <rollingfile type="size" filename="` + logFile + `" maxsize="` + strconv.Itoa(logFileMax) + `" maxrolls="` + strconv.Itoa(logBackups) + `"/>
//    </outputs>
//    <formats>
//        <format id="colored"  format="%Time %EscM(46)%Level%EscM(49) %Msg%n%EscM(0)"/>
//        <format id="common" format="%Date/%Time [%LEV] [%File:%Line] %Msg%n" />
//        <format id="critical" format="%Date/%Time %File:%Line %Func %Msg%n" />
//    </formats>
//</seelog>
//`
//
//	logger, _ := log.LoggerFromConfigAsBytes([]byte(LogConfig))
//	log.UseLogger(logger)
//}


//头部信息显示显示
func (pdo *Pdo) displayHeader() {
	//头部输出
		fmt.Println(">>>> Welcome " + SUDO_USER + "...")
		for i, elem := range pdo.JobList {
			fmt.Printf("%-25s-%-20s ", elem.Host, elem.Path)
			if (i+1)%2 == 0 {
				fmt.Printf("\n")
			}
		}
		fmt.Printf("\n")
		fmt.Println("#--Total--# ", pdo.Jobsinfo.Total)
		fmt.Println("#---CMD---# ", pdo.Command.Display)

	}


//尾部信息显示
func (pdo *Pdo)displaySummary() {

	info:=pdo.Jobsinfo

	fmt.Printf("\n[INFO] Total:%d Success:%d Fail:%d TimeOver:%d Connect:%d\n",info.Total,info.Success,info.ExeFail,info.TimeOver,info.ConnectFail)
}

//检查文件是否存在.
func checkExist(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil || os.IsExist(err)
}

func (pdo *Pdo)jobstring(job *Job) string {
	var jobstring string
	if job.jobname.User != USERNAME  || job.jobname.Port != DefaultSSHPort {
	    jobstring = fmt.Sprintf("%s@%s:%s  %s",job.jobname.User,job.jobname.Host,job.jobname.Port,job.jobname.Path)
	}else{
		jobstring = fmt.Sprintf("%s  %s",job.jobname.Host,job.jobname.Path)
		}
	return jobstring
}

//具体job处理过程
func (pdo *Pdo) Do(job *Job) {

	var apperr *appError
	jobstring := pdo.jobstring(job)

	if apperr=pdo.PreDo(job,pdo.Command.PreCmd) ; apperr!=nil{
		job.results <- JobResult{jobstring, apperr.Code, apperr.Message,apperr.Error.Error()}
		return
	}


	if pdo.Command.Local {
		apperr=pdo.ExeLocalCmd(job,pdo.Command.Execmd,false)
	}else{
		apperr=pdo.ExeRemoteCmd(job,pdo.Command.Execmd,false)
	}

	if apperr!=nil {
		job.results <- JobResult{jobstring, apperr.Code, apperr.Message,apperr.Error.Error()}
	}
	pdo.PostDo()
}


func (pdo *Pdo)PreDo(job *Job,cmd string) *appError{
	if pdo.Command.PreCmd == "" {
		return nil
	}
	return pdo.ExeLocalCmd(job ,cmd,true)
}

func (pdo *Pdo)PostDo() error {
	if pdo.Command.PostCmd == "" {
		return nil
	}

	return nil
}
