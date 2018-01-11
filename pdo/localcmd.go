package pdo

import (
//	"fmt"
	"html/template"
	"bytes"
	"os/exec"
	"time"
	"io"
//	"log"
	"syscall"
//	"fmt"
)

func (pdo *Pdo)ParseCmd(job HostList)(string, error) {
	tmpl, err := template.New("pdo").Parse(pdo.Command.Inputcmd)
	var tempBuf bytes.Buffer
	err = tmpl.Execute(&tempBuf, job)
	ExecCmd := tempBuf.String()
	return  ExecCmd, err
}

func (pdo *Pdo)ExeLocalCmd(job *Job,quiet bool)error{

	var out ,outerr bytes.Buffer
	var result JobResult
	var execCmd string
	var err error

	list:=job.jobname
	if pdo.Command.Local {
		execCmd,err=pdo.ParseCmd(list)
		if err!=nil {
			return err
		}
	}

	cmd := exec.Command("/bin/bash", "-s")
	cmd.Stdin = bytes.NewBufferString(execCmd)


	jobstring := pdo.jobstring(job)

	//命令执行
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err!=nil {
		return err
	}

	if len(pdo.Output.Save) != 0 {
		for k,v:=range pdo.Output.Save {
			if k == OutDirectory {
				if err:=pdo.OutputSaveToDirectory(job.jobname.Host,v,job.jobid,stdout,stderr); err!=nil {
					return err
				}
			}
			if k == OutFile || k == OutFileAppend{
				if err:=pdo.OutputFormatRow(job.jobname.Host,stdout,pdo.Output.Regex,true); err!=nil{
					return  err
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
	if !quiet {
		job.results <- result
	}

	return nil
}

