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
	Close   chan struct{}
}

func (wc *WorkerConnection) SendCommand(cmd string) (*command.CommandResponse, error) {
	var buff bytes.Buffer
	res := command.NewCommandResponse(wc.host.Hostname)
	wc.session.Stdout = &buff
	if err := wc.session.Run(cmd); err != nil {
		fmt.Println(wc.host.Hostname, "failed to run:", cmd, err.Error())
		return res, err
	}
	res.Data = bytes.Replace(buff.Bytes(), []byte("\x00"), []byte{}, -1)
	return res, nil
}

func (wc *WorkerConnection) Start(host WorkerHost) (chan struct{}, error) {
	wc.Close = make(chan struct{})
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

func Stop() {

}

func NewConnectedWorker(workerHost WorkerHost) (*WorkerConnection, error) {
	conn := WorkerConnection{}
	_, err := conn.Start(workerHost)
	if err != nil {
		fmt.Println("Error connecting to", workerHost.Hostname)
		return &conn, err
	}
	return &conn, nil
}

func NewConnectededWorkerBatch(workerHosts []WorkerHost) ([]*WorkerConnection, []*WorkerConnection) {
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
			conn, err := NewConnectedWorker(host)
			if err != nil {
				return
			}
			conPipeline <- conn
		}(host)
	}
	wg.Wait()
	close(conPipeline)
	return workerConns, workerConnErrs
}
