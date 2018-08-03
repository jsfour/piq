package worker

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jsmootiv/piq/command"
	"golang.org/x/crypto/ssh"
)

type WorkerConnection struct {
	hostname string
	user     string
	password string
	session  *ssh.Session
}

func (wc *WorkerConnection) SendCommand(cmd string) (*command.CommandResponse, error) {
	var buff bytes.Buffer
	res := command.NewCommandResponse(wc.hostname)
	wc.session.Stdout = &buff
	if err := wc.session.Run(cmd); err != nil {
		fmt.Println("Failed to run: " + cmd + err.Error())
		return res, err
	}
	res.Data = bytes.Replace(buff.Bytes(), []byte("\x00"), []byte{}, -1)
	return res, nil
}

func (wc *WorkerConnection) Start(fullHost string) (chan struct{}, error) {
	quit := make(chan struct{})
	addySlit := strings.Split(fullHost, "@")
	wc.password = addySlit[0]
	wc.hostname = addySlit[1] + ":22"
	wc.user = "root"
	fmt.Println("Connecting to:", wc.hostname)
	config := &ssh.ClientConfig{
		User: wc.user,
		Auth: []ssh.AuthMethod{
			ssh.Password(wc.password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1 * time.Second,
	}
	client, err := ssh.Dial("tcp", wc.hostname, config)
	if err != nil {
		return quit, err
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return quit, err
	}

	wc.session = session

	go func() {
		defer session.Close()
		<-quit
	}()

	return quit, nil
}

func NewConnectededWorkers(workerHosts []string) []*WorkerConnection {
	var wg sync.WaitGroup
	var workerConns []*WorkerConnection
	conPipeline := make(chan *WorkerConnection)

	go func() {
		for conn := range conPipeline {
			workerConns = append(workerConns, conn)
		}
	}()

	for _, workerAddress := range workerHosts {
		wg.Add(1)
		go func(workerAddress string) {
			conn := WorkerConnection{}
			conn.Start(workerAddress)
			conPipeline <- &conn
			wg.Done()
		}(workerAddress)
	}
	wg.Wait()
	close(conPipeline)
	return workerConns
}
