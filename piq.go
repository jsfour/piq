package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"

	"github.com/jsmootiv/piq/command"
	"github.com/jsmootiv/piq/worker"
	"github.com/spf13/cobra"
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

func startPrinter(quit chan struct{}, inputFeed chan command.CommandResponse) chan struct{} {
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

func getStats(cmd *cobra.Command, args []string) {
	fmt.Println("Starting collection")
	appCfg, err := OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	workers, err := worker.NewWorkerHostBatch(appCfg.Workers)
	if err != nil {
		fmt.Println("Failed to read workers: ", err)
		panic(1)
	}

	quit := make(chan struct{})
	defer close(quit)

	workerConns := worker.NewConnectededWorkers(workers)
	responseFeed := make(chan command.CommandResponse)
	printerDone := startPrinter(quit, responseFeed)

	for _, conn := range workerConns {
		cmd := command.NewSummaryCommand()
		res, err := conn.SendCommand(cmd)
		if err != nil {
			res.Data = command.NewErrorJson(err)
			continue
		}
		responseFeed <- *res
	}
	close(responseFeed)
	<-printerDone
}

func killWorker(cmd *cobra.Command, args []string) {
	fmt.Println("Killing worker not supported yet")
	// appCfg, err := OpenConfig("./config.json")
	// if err != nil {
	// 	fmt.Println("Failed to load config: ", err)
	// 	panic(1)
	// }
	// var targetWorkers string
	// for _, worker := appCfg.Workers {
	// 	if
	// }
}

func main() {
	var stats = &cobra.Command{
		Use:   "stats",
		Short: "Pulls stats from workers",
		Run:   getStats,
	}

	var kill = &cobra.Command{
		Use:   "kill [hostname]",
		Short: "Kills worker",
		Run:   killWorker,
	}

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(stats)
	rootCmd.AddCommand(kill)
	rootCmd.Execute()
}
