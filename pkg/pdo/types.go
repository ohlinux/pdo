package pdo

import (
	"os"
	"time"
	"strconv"
	"path/filepath"
)

var (
	SUDO_USER  = os.Getenv("SUDO_USER")
	USERNAME   = os.Getenv("USER")
	HOME       = os.Getenv("HOME")
	USERTMPDIR = "/tmp/pdo_" + USERNAME
	PID        = strconv.Itoa(os.Getppid())
	FAILF 	   = filepath.Join(USERTMPDIR,"fail."+PID)
	SUCCESSF   = filepath.Join(USERTMPDIR,"success."+PID)
)

const (
	ResultSuccess        = 0
	ResultStartFailed    = 1
	ResultTimeOverKilled = 2
	ResultKillFailed     = 3
	ResultConnectFailed  = 255

	FormatRow  = "row" // have host for each line  when doing
	FormatJson = "json" // command finish
	FormatYaml = "yaml" // command finish
	FormatText = "text" // is default format when command finish

	ErrorTimeOverKillFailed = "Time over, killed..."

	StatusSuccess = "SUCCESS"
	StatusTimeOverKilled = "Time Over KILLED"
	StatusTimeOverKilledFailed = "KILLED FAILED"
	StatusConnectFailed = "Connect FAILED"
	StatusDoFailed = "FAILED"

	DefaultSSHPort = "22"


	OutDirectory="directory"
	OutFile="file"
	OutFileAppend="file-append"
)


 type PdoLog struct {
 	Directory string `json:"directory" yaml:"directory"`
 	Backup int `json:"backup" yaml:"backup"`
 	Maxfile int `json:"maxfile" yaml:"maxfile"`
 	Level string `json:"level" yaml:"level"`
 }
 
 type PdoInput struct {
 	From string `json:"from"`
 	Format string `json:"format"`
 	Regex string `json:"regex"`
 }
 
 type PdoOutput struct {
 	Format string `json:"format"`
 	Save   map[string]string `json:"save"` // if Save is nil or len(save) ==0 output to screen
 	Regex string `json:"regex"`
 	Nocolor bool `json:"nocolor"`
 	Summary bool `json:"summary"`
 	Header bool `json:"header"`
 	File *os.File
 }
 
 type PdoParallel struct {
 	Numbers int `json:"numbers"`
 	OverTime time.Duration `json:"overTime" yaml:"overTime"`
 	IntervalTime time.Duration `json:"intervalTime" yaml:"intervalTime"`
 	Yes bool  `json:"yes"`
 	Ask int  `json:"ask"`
 	Pause bool `json:"pause"`
 }

 type PdoAuth struct {
 	PrivateKey string `json:"privateKey" yaml:"privateKey"`
 //	KnownHosts string `json:"knownHosts" yaml:"knownHosts"`
 	User string `json:"user"`
 	Passwd string `json:"passwd"`
 	Port   string `json:"port"`
 }

type Pdo struct {
	Log PdoLog   	`json:"log"`
	Template []map[string]string `json:"template"`
	Shortcmd  []map[string]string `json:"shortcmd"`
	Plugins  []string	 `json:"plugins"`
	Input    PdoInput	`json:"input"`
	Output   PdoOutput	`json:"output"`
	Auth    PdoAuth	`json:"auth"`
	Parallel PdoParallel `json:"parallel"`
	JobList  []HostList	`json:"jobList" yaml:"jobList"`
	Jobsinfo      Jobsinfo `json:"jobsinfo"`
	Command   Command `json:"command"`
	Args       []string `json:"args"`
	WorkEnv   WorkEnv
}

type WorkEnv struct {
	FailFile *os.File
	SuccessFile *os.File
}

type Command struct {
	Display string `json:"display"`
	Inputcmd    string `json:"inputcmd"`
	PreCmd   string `json:precmd`
	Execmd   string `json:"execmd"`
	PostCmd string  `json:"postcmd"`
	Args   []string `json:"args"`
    Copy    string  `json:"copy"`
    Local    bool `json:"local"`
    Kill    bool `json:"kill" description:"remoted will kill pid when ctrl+c ,eg: tail -f"`
}

type Jobsinfo struct {
	Finished   int
	Success int
	Total      int
	ExeFail       int
	TimeOver    int
	ConnectFail  int
}

type Job struct {
	jobname HostList
	results chan<- JobResult
	jobid   int
}

type JobResult struct {
	Jobname    string `json:"jobname"`
	RetCode    int   `json:"retcode"`
	Stdout     string  `json:"stdout"`
	Stderr     string  `json:"stderr"`
}

type HostList struct {
	Host string `json:"host"`
	Path string `json:"path"`
	User string `json:"user"` 
	Port string `json:"port"`
	Passwd string `json:"passwd"`
}

type appError struct {
	Error   error
	Message string
	Code    int
}
