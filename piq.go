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

	"github.com/jsmootiv/piq/command"
	"github.com/spf13/cobra"
	"golang.org/x/crypto/ssh"
)

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

func startPrinter(quit chan struct{}, inputFeed chan commandResponse) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for res := range inputFeed {
			select {
			case <-quit:
				return
			default:
				var myRes command.SummaryRes
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

			cmd := command.NewSummaryCommand()
			if err := session.Run(cmd); err != nil {
				fmt.Println("Failed to run: " + cmd + err.Error())
				panic(1)
			}

			res.Data = bytes.Replace(buff.Bytes(), []byte("\x00"), []byte{}, -1)
			responseFeed <- *res

		}(workerAddress)
	}
	wg.Wait()
	close(responseFeed)
	<-printerDone
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
