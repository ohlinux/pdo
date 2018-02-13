package pdo

import (
	"html/template"
	"bytes"
	"os/exec"
	"time"
	"io"
//	"log"
	"syscall"
//	"fmt"
)

func (pdo *Pdo)ParseCmd(job HostList,cmd string)(string, error) {
	tmpl, err := template.New("pdo").Parse(cmd)
	var tempBuf bytes.Buffer
	err = tmpl.Execute(&tempBuf, job)
	ExecCmd := tempBuf.String()
	return  ExecCmd, err
}

func (pdo *Pdo)ExeLocalCmd(job *Job,command string,quiet bool)*appError{

	var out ,outerr bytes.Buffer
	var result JobResult
	var execCmd string
	var err error

	list:=job.jobname
		execCmd,err=pdo.ParseCmd(list,command)
		if err!=nil {
			return &appError{err,"Can't parse command",ResultStartFailed}
		}


	cmd := exec.Command("/bin/bash", "-s")
	cmd.Stdin = bytes.NewBufferString(execCmd)


	jobstring := pdo.jobstring(job)

	//命令执行
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return &appError{err,"Can't create stdout",ResultStartFailed}
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return &appError{err,"Can't create stderr",ResultStartFailed}
	}

	err = cmd.Start()
	if err!=nil {
		return &appError{err,"Can't start to run command",ResultStartFailed}
	}

	if len(pdo.Output.Save) != 0 {
		for k,v:=range pdo.Output.Save {
			if k == OutDirectory {
				if err:=pdo.OutputSaveToDirectory(job.jobname.Host,v,job.jobid,stdout,stderr); err!=nil {
					return &appError{err,"Can't save output to directory",ResultStartFailed}
				}
			}
			if k == OutFile || k == OutFileAppend{
				if err:=pdo.OutputFormatRow(job.jobname.Host,stdout,pdo.Output.Regex,true); err!=nil{
					return &appError{err,"Can't show output row by row ",ResultStartFailed}
				}
			}
		}
	}else {
		if pdo.Output.Format == FormatRow {
			pdo.OutputFormatRow(job.jobname.Host, stdout, pdo.Output.Regex,false)
		} else {
				go io.Copy(&out, stdout)
				go io.Copy(&outerr, stderr)
		}
	}

	done := make(chan error)

	go func() {
		done <- cmd.Wait()
	}()

	//线程控制执行时间
	select {
	case <-time.After(pdo.Parallel.OverTime):
		//超时被杀时
		if err := cmd.Process.Kill(); err != nil {
			//超时被杀失败
			result.RetCode=ResultKillFailed
			result.Stderr=err.Error()
		}else{
			result.RetCode=ResultTimeOverKilled
			result.Stderr=ErrorTimeOverKillFailed
		}
		result.Stdout=out.String()
		<-done
	case err := <-done:
		if err != nil {
			var exitCode int
			if exiterr, ok := err.(*exec.ExitError); ok {
				if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
				}
			}
			result.RetCode=exitCode
		} else {
			result.RetCode=ResultSuccess
		}

		result.Stderr=outerr.String()
		result.Stdout=out.String()
	}

	result.Jobname=jobstring

	if quiet {
		if result.RetCode != ResultSuccess{
		return nil
			//return &appError{fmt.Errorf("run failed, %+v\n",execCmd),"Can't show output row by row ",ResultStartFailed}
		}
	}

	job.results <- result
	return nil
}

