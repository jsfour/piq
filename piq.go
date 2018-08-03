package main

import (
	"bufio"
	"bytes"
	"crypto"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

// killed clients
// root@192.168.4.150
// root@192.168.4.152
// root@192.168.4.156
// root@192.168.4.157
// root@192.168.4.158

type config struct {
	Workers []string `json:"workers"`
}

func OpenConfig(location string) (*config, error) {
	cfg := &config{}
	jsonFile, err := os.Open(location)

	if err != nil {
		currUser, err := user.Current()
		if err != nil {
			return cfg, nil
		}
		userCfg := currUser.HomeDir + "/.piq/config.json"
		if location == userCfg {
			return cfg, errors.New("No config found")
		}
		return OpenConfig(userCfg)
	}
	defer jsonFile.Close()

	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		return cfg, nil
	}
	json.Unmarshal(byteValue, cfg)
	return cfg, nil
}

func hostKeyCheck(hostname string, remote net.Addr, key crypto.PublicKey) error {
	// Every client must provide a host key check.  Here is a
	// simple-minded parse of OpenSSH's known_hosts file
	file, err := os.Open(filepath.Join(os.Getenv("HOME"), ".ssh", "known_hosts"))
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var hostKey ssh.PublicKey
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), " ")
		if len(fields) != 3 {
			continue
		}
		if strings.Contains(fields[0], hostname) {
			var err error
			hostKey, _, _, _, err = ssh.ParseAuthorizedKey(scanner.Bytes())
			if err != nil {
				fmt.Println("error parsing %q: %v", fields[2], err)
				panic(1)
			}
			break
		}
	}

	if hostKey == nil {
		// TODO: allow to add if nil
		return nil
	}

	return nil
}

type commandResponse struct {
	Data   []byte
	Source string
}

func NewCommandResponse(source string) *commandResponse {
	return &commandResponse{
		Data:   []byte{},
		Source: source,
	}
}

type status struct {
	// TODO: move me
	Status      string `json:"STATUS"`
	When        int64  `json:"When"`
	Code        int64  `json:"Code"`
	Msg         string `json:"Msg"`
	Description string `json:"Description"`
}

type summary struct {
	// TODO: move me
	Elapsed            int     `json:"Elapsed"`
	Ghs5s              string  `json:"GHS 5s"`
	GhsAv              float64 `json:"GHS av"`
	FoundBlocks        int     `json:"Found Blocks"`
	Getworks           int     `json:"Getworks"`
	Accepted           int     `json:"Accepted"`
	Rejected           int     `json:"Rejected"`
	HardwareErrors     int     `json:"Hardware Errors"`
	Utility            float64 `json:"Utility"`
	Discarded          int     `json:"Discarded"`
	Stale              int     `json:"Stale"`
	GetFailures        int     `json:"Get Failures"`
	LocalWork          int     `json:"Local Work"`
	RemoteFailures     int     `json:"Remote Failures"`
	NetworkBlocks      int     `json:"Network Blocks"`
	TotalMh            float64 `json:"Total MH"`
	WorkUtility        float64 `json:"Work Utility"`
	DifficultyAccepted float64 `json:"Difficulty Accepted"`
	DifficultyStale    float64 `json:"Difficulty Stale"`
	DifficultyRejected float64 `json:"Difficulty Rejected"`
	BestShare          int     `json:"Best Share"`
	DeviceHardwarePerc float64 `json:"Device Hardware%"`
	DeviceRejectedPerc float64 `json:"Device Rejected%"`
	PoolRejectedPerc   float64 `json:"Pool Rejected%"`
	PoolStalePerc      float64 `json:"Pool Stale%"`
	Lastgetwork        int     `json:"Last getwork"`
}

type summaryRes struct {
	Status  []status  `json:"STATUS"`
	Summary []summary `json:"SUMMARY"`
	Error   string    `json:"error";omitempty`
}

func startPrinter(quit chan struct{}, inputFeed chan commandResponse) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for res := range inputFeed {
			select {
			case <-quit:
				return
			default:
				var myRes summaryRes
				err := json.Unmarshal(res.Data, &myRes)
				if err != nil {
					fmt.Println(res.Source, "Error parsing", err)
					continue
				}
				if myRes.Error != "" {
					fmt.Println(res.Source, "Error", myRes.Error)
					continue
				}
				fmt.Println(res.Source)
				fmt.Println("        Average  Hashrate", myRes.Summary[0].GhsAv)
				fmt.Println("        5sec  	  Hashrate", myRes.Summary[0].Ghs5s)
				fmt.Println("        Hardware Errors", myRes.Summary[0].HardwareErrors)

			}
		}
	}()
	return done
}

func newErrorJson(err error) string {
	return fmt.Sprint(`{ "error": "`, err.Error(), `"}`)
}

func getStats(cmd *cobra.Command, args []string) {
	var wg sync.WaitGroup
	fmt.Println("Starting collection")
	appCfg, err := OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	workers := appCfg.Workers

	quit := make(chan struct{})
	responseFeed := make(chan commandResponse)
	printerDone := startPrinter(quit, responseFeed)
	defer close(quit)

	for _, workerAddress := range workers {
		wg.Add(1)
		go func(workerAddress string) {
			addySlit := strings.Split(workerAddress, "@")
			workersPass := addySlit[0]
			workerHost := addySlit[1] + ":22"
			fmt.Println("Getting data from:", workerHost)
			defer wg.Done()
			res := NewCommandResponse(workerHost)
			config := &ssh.ClientConfig{
				User: "root",
				Auth: []ssh.AuthMethod{
					ssh.Password(workersPass),
				},
				HostKeyCallback: ssh.InsecureIgnoreHostKey(),
				Timeout:         1 * time.Second,
			}
			client, err := ssh.Dial("tcp", workerHost, config)
			if err != nil {
				msg := newErrorJson(err)
				res.Data = []byte(msg)
				responseFeed <- *res
				return
			}

			// Each ClientConn can support multiple interactive sessions,
			// represented by a Session.
			session, err := client.NewSession()
			if err != nil {
				fmt.Println("Failed to create session: ", err)
				panic(1)
			}
			defer session.Close()

			// Once a Session is created, you can execute a single command on
			// the remote side using the Run method.
			var buff bytes.Buffer
			session.Stdout = &buff
			if err := session.Run(`echo '{"command": "summary"}' | nc localhost 4028`); err != nil {
				fmt.Println("Failed to run: " + err.Error())
				panic(1)
			}

			res.Data = bytes.Replace(buff.Bytes(), []byte("\x00"), []byte{}, -1)
			responseFeed <- *res
		}(workerAddress)
	}
	wg.Wait()
	close(responseFeed)
	<-printerDone
	fmt.Println("Complete")
}

func main() {
	var stats = &cobra.Command{
		Use:   "stats",
		Short: "Pulls stats from workers",
		Run:   getStats,
	}

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(stats)
	rootCmd.Execute()
}
