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
	Host    WorkerHost
	session *ssh.Session
	Close   chan struct{}
}

func (wc *WorkerConnection) SendCommand(cmd string) (*command.CommandResponse, error) {
	var buff bytes.Buffer
	res := command.NewCommandResponse(wc.Host.Hostname)
	wc.session.Stdout = &buff
	if err := wc.session.Run(cmd); err != nil {
		fmt.Println(wc.Host.Hostname, "failed to run:", cmd, err.Error())
		res.Status = command.Failed
		return res, err
	}
	res.Data = bytes.Replace(buff.Bytes(), []byte("\x00"), []byte{}, -1)
	res.Status = command.Success
	return res, nil
}

func (wc *WorkerConnection) Start(host WorkerHost) (chan struct{}, error) {
	wc.Close = make(chan struct{})
	wc.Host = host
	config := &ssh.ClientConfig{
		User: wc.Host.Username,
		Auth: []ssh.AuthMethod{
			ssh.Password(wc.Host.Password),
		},
		// TODO: fix the host key callback
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         1 * time.Second,
	}
	client, err := ssh.Dial("tcp", wc.Host.Hostname, config)
	if err != nil {
		return wc.Close, err
	}

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := client.NewSession()
	if err != nil {
		fmt.Println("Failed to create session: ", err)
		return wc.Close, err
	}

	wc.session = session

	go func() {
		defer session.Close()
		<-wc.Close
	}()

	return wc.Close, nil
}

func NewConnectedWorker(workerHost WorkerHost) (*WorkerConnection, error) {
	conn := WorkerConnection{}
	_, err := conn.Start(workerHost)
	if err != nil {
		return &conn, err
	}
	return &conn, nil
}

func NewConnectededWorkerBatch(workerHosts []WorkerHost) ([]*WorkerConnection, []WorkerHost) {
	var wg sync.WaitGroup
	var workerConns []*WorkerConnection
	var hostsNotConnected []WorkerHost

	conPipeline := make(chan *WorkerConnection)
	errPipeline := make(chan WorkerHost)

	go func() {
		for conn := range conPipeline {
			workerConns = append(workerConns, conn)
		}
	}()

	go func() {
		for host := range errPipeline {
			hostsNotConnected = append(hostsNotConnected, host)
		}
	}()

	for _, host := range workerHosts {
		wg.Add(1)
		go func(host WorkerHost) {
			defer wg.Done()
			conn, err := NewConnectedWorker(host)
			if err != nil {
				errPipeline <- host
				return
			}
			conPipeline <- conn
		}(host)
	}
	wg.Wait()
	close(conPipeline)
	close(errPipeline)

	return workerConns, hostsNotConnected
}
