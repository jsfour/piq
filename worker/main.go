package worker

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/jsmootiv/piq/command"
	"golang.org/x/crypto/ssh"
)

type WorkerConnection struct {
	host    WorkerHost
	session *ssh.Session
}

func (wc *WorkerConnection) SendCommand(cmd string) (*command.CommandResponse, error) {
	var buff bytes.Buffer
	res := command.NewCommandResponse(wc.host.Hostname)
	wc.session.Stdout = &buff
	if err := wc.session.Run(cmd); err != nil {
		fmt.Println(wc.host.Hostname, " failed to run: ", cmd, err.Error())
		return res, err
	}
	res.Data = bytes.Replace(buff.Bytes(), []byte("\x00"), []byte{}, -1)
	return res, nil
}

func (wc *WorkerConnection) Start(host WorkerHost) (chan struct{}, error) {
	quit := make(chan struct{})
	wc.host = host
	fmt.Println(host.Hostname, " Connecting")
	config := &ssh.ClientConfig{
		User: wc.host.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(wc.host.Password),
		},
		// TODO: fix the host key callback
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1 * time.Second,
	}
	client, err := ssh.Dial("tcp", wc.host.Hostname, config)
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

func NewConnectededWorkers(workerHosts []WorkerHost) ([]*WorkerConnection, []*WorkerConnection) {
	var wg sync.WaitGroup
	var workerConns []*WorkerConnection
	var workerConnErrs []*WorkerConnection

	conPipeline := make(chan *WorkerConnection)

	go func() {
		for conn := range conPipeline {
			workerConns = append(workerConns, conn)
		}
	}()

	for _, host := range workerHosts {
		wg.Add(1)
		go func(host WorkerHost) {
			defer wg.Done()
			conn := WorkerConnection{}
			_, err := conn.Start(host)
			if err != nil {
				fmt.Println("Error connecting to", host.Hostname)
				return
			}
			conPipeline <- &conn
		}(host)
	}
	wg.Wait()
	close(conPipeline)
	return workerConns, workerConnErrs
}
