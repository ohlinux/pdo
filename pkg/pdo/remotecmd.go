package pdo

import (
//	"bufio"
//	"errors"
	"fmt"
	"io/ioutil"
	"log"
	//"os"
	//"path/filepath"
	//"strings"
	"time"
	"text/template"
	"bytes"
	"io"
	"os/exec"
	"syscall"

	"golang.org/x/crypto/ssh"
)

func PublicKeyFile(file string) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	key, err := ssh.ParsePrivateKey(buffer)
	if err != nil {
		return nil
	}
	return ssh.PublicKeys(key)
}

func (pdo *Pdo)connect(user, password, host string, port string ) (*ssh.Session, error) {
	var (
		auth         []ssh.AuthMethod
		addr         string
		clientConfig *ssh.ClientConfig
		client       *ssh.Client
		session      *ssh.Session
		err          error
	)

	if user == "" {
		user = "root"
	}
		//var hostKey ssh.PublicKey
		//hostKey, err = pdo.getHostKey(host)
		//if err != nil {
		//	return nil,err
		//}

	if password != "" {
		// get auth method
		auth = make([]ssh.AuthMethod, 0)
		auth = append(auth, ssh.Password(password))

		clientConfig = &ssh.ClientConfig{
			User: user,
			Auth: auth,
			Timeout: time.Second * 3,
			//HostKeyCallback: ssh.FixedHostKey(hostKey),
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			//HostKeyCallback: func(hostname string, remote net.Addr, key ssh.PublicKey) error {
			//	fmt.Printf("%s %s %s\n", strings.Split(hostname, ":")[0], key.Type(), base64.StdEncoding.EncodeToString(key.Marshal()))
			//	return nil
			//},
		}
	} else {
		//var hostKey ssh.PublicKey
		//hostKey, err := pdo.getHostKey(host)
		//if err != nil {
		//	log.Fatal(err)
		//}
		var privateKey string
		if pdo.Auth.PrivateKey == "" {
			privateKey=HOME+".ssh/id_rsa"
		}else{
			privateKey=pdo.Auth.PrivateKey
		}

		key, err := ioutil.ReadFile(privateKey)
		if err != nil {
			log.Fatalf("unable to read private key: %v", err)
		}

		// Create the Signer for this private key.
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			log.Fatalf("unable to parse private key: %v", err)
		}

		clientConfig = &ssh.ClientConfig{
			User: user,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			Timeout:         3 * time.Second,
			HostKeyCallback: ssh.InsecureIgnoreHostKey(),
			//HostKeyCallback: ssh.FixedHostKey(hostKey),
		}

	}
	// connet to ssh
	addr = fmt.Sprintf("%s:%s", host, port)
	if client, err = ssh.Dial("tcp", addr, clientConfig); err != nil {
		return nil, err
	}

	// create session
	if session, err = client.NewSession(); err != nil {
		return nil, err
	}

	return session, nil
}

func (pdo *Pdo)ExeRemoteCmd(job *Job,quiet bool)error{
	var out ,outerr bytes.Buffer
	var result JobResult
	var err error

	list:=job.jobname

	session, err := pdo.connect(list.User, list.Passwd, list.Host, list.Port)
	if err!=nil {
		return err
	}
	defer session.Close()

	jobstring := pdo.jobstring(job)
	stdout, err := session.StdoutPipe()
	if err != nil {
		return err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		return err
	}

	if pdo.Command.Remoted {
		err = session.Start(shellCmd(pdo.Command.Execmd))
	}else{
		err = session.Start(pdo.Command.Execmd)
	}

	if err!=nil {
		return err
	}

	////直接输出
	go io.Copy(&out, stdout)
	go io.Copy(&outerr, stderr)

	done := make(chan error)

	go func() {
		done <- session.Wait()
	}()

	//线程控制执行时间
	select {
	case <-time.After(pdo.Parallel.OverTime):
		//超时被杀时
		if err := session.Signal(ssh.SIGKILL); err != nil {
			//超时被杀失败
			result.RetCode=ResultKillFailed
			result.Stderr=err.Error()
		}else{
			result.RetCode=ResultTimeOverKilled
			result.Stderr="Time over, killed..."
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

/*
func (pdo *Pdo)getHostKey(host string) (ssh.PublicKey, error) {
	var knownhosts string
	if pdo.Auth.KnownHosts == "" {
		knownhosts=filepath.Join(HOME, ".ssh", "known_hosts")
	}else{
		knownhosts=pdo.Auth.KnownHosts
	}

	file, err := os.Open(knownhosts)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		//不精确 会有问题
		if strings.Contains(fields[0], host) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error parsing %q: %v", fields[2], err))
			}
			break
		}
	}

	if hostKey == nil {
		return nil, errors.New(fmt.Sprintf("no hostkey for %s", host))
	}
	return hostKey, nil
}
*/

type Cmd struct {
	Param string
}

func shellCmd(cmd string) string {
	params := Cmd{cmd}
	//tmpl, err := template.ParseFiles("remote.sh")
	tmpl,err:=template.New("remote").Parse(remote)
	if err != nil {
		log.Fatal(err)
	}
	var tpl bytes.Buffer
	err=tmpl.Execute(&tpl,params)
	//err := tmpl.ExecuteTemplate(&tpl, "remote.sh", params)
	if err != nil {
		log.Fatal(err)
	}

	return tpl.String()

}

var remote string = `
#!/bin/bash
function get_ppid {
	THE_PPID=$1
	awk '/PPid:/ {print $2;}' /proc/$THE_PPID/status
}
function get_ppid_name {
	THE_PPID=$1
	awk '/Name:/ {print $2;}' /proc/$THE_PPID/status
}
{{.Param}}&
CHILD_PID=$!
CUR_PPID=$PPID
THE_PPPID=$(get_ppid $CUR_PPID)
THE_PPPID_NAME=$(get_ppid_name $CUR_PPID)
while [ "$THE_PPPID_NAME" != "sshd" ] && (($THE_PPPID > 2)); do
	CUR_PPID=$THE_PPPID
	THE_PPPID=$(get_ppid $CUR_PPID)
	THE_PPPID_NAME=$(get_ppid_name $CUR_PPID)
done
while true; do
	if [ ! -e /proc/$CHILD_PID ]; then
		exit
	fi
	if [ ! -e /proc/$CUR_PPID ]; then
		GROUP_ID=$(ps x -o "%p %r" | awk -v self="$$" '{if ($1 == self) print $2;}')
		kill  -$GROUP_ID
		exit
	fi
	usleep 100000
done
`
