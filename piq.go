package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"strings"

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

func startRawPrinter(quit chan struct{}, inputFeed chan command.CommandResponse) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for res := range inputFeed {
			select {
			case <-quit:
				return
			default:
				fmt.Println(res.Source)
				fmt.Println("     ", fmt.Sprintf("%s", res.Data))
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

	// TODO: Process error connections
	workerConns, failedHosts := worker.NewConnectededWorkerBatch(workers)
	responseFeed := make(chan command.CommandResponse)
	printerDone := startPrinter(quit, responseFeed)

	for _, host := range failedHosts {
		res := command.CommandResponse{
			Data:   []byte("Error Connecting"),
			Source: host.Hostname,
		}
		responseFeed <- res
	}

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

func scaleWorkerUp(cmd *cobra.Command, args []string) {
	fmt.Println("Worker scaling not supported")
	// targetWorker := args[0]
	// fmt.Println("Scaling up worker", targetWorker)
	// appCfg, err := OpenConfig("./config.json")
	// if err != nil {
	// 	fmt.Println("Failed to load config: ", err)
	// 	panic(1)
	// }
	// targetWorker = targetWorker + ":22"

	// for _, rawWorkerHostname := range appCfg.Workers {
	// 	currentWorker, err := worker.NewWorkerHostFromRaw(rawWorkerHostname)
	// 	if err != nil {
	// 		fmt.Println("Not able to scale worker")
	// 		panic(1)
	// 	}

	// 	if currentWorker.Hostname != targetWorker {
	// 		continue
	// 	}

	// 	for i := 0; i < 3; i++ {
	// 		fmt.Println("    pool", i)
	// 		// HACK: I dont need to reconnect every time
	// 		// TODO: consolidate with the scaleUp

	// 		workerConn, _ := worker.NewConnectedWorker(currentWorker)
	// 		scaleCommand := command.NewScaleUpCommand(i)
	// 		res, err := workerConn.SendCommand(scaleCommand)

	// 		if err != nil {
	// 			res.Data = command.NewErrorJson(err)
	// 			continue
	// 		}
	// 		fmt.Printf("%s\n", res.Data)
	// 		close(workerConn.Close)
	// 	}
	// }
}

func scaleDownWorker(cmd *cobra.Command, args []string) {
	fmt.Println("Worker scaling not supported")
	// targetWorker := args[0]
	// appCfg, err := OpenConfig("./config.json")
	// if err != nil {
	// 	fmt.Println("Failed to load config: ", err)
	// 	panic(1)
	// }
	// targetWorker = targetWorker + ":22"

	// for _, rawWorkerHostname := range appCfg.Workers {
	// 	currentWorker, err := worker.NewWorkerHostFromRaw(rawWorkerHostname)
	// 	if err != nil {
	// 		fmt.Println("Not able to scale worker")
	// 		panic(1)
	// 	}

	// 	if currentWorker.Hostname != targetWorker {
	// 		continue
	// 	}

	// 	for i := 0; i < 3; i++ {
	// 		fmt.Println("    pool", i)
	// 		// HACK: I dont need to reconnect every time
	// 		// TODO: consolidate with the scaleDown

	// 		workerConn, _ := worker.NewConnectedWorker(currentWorker)
	// 		cmd := command.NewScaleDownCommand(i)
	// 		fmt.Println("%v", cmd)

	// 		res, err := workerConn.SendCommand(cmd)

	// 		if err != nil {
	// 			res.Data = command.NewErrorJson(err)
	// 			continue
	// 		}
	// 		fmt.Printf("    %s\n", res.Data)
	// 		close(workerConn.Close)
	// 	}
	// }
}

func killWorker(cmd *cobra.Command, args []string) {
	targetWorker := strings.ToLower(args[0])
	var killList []worker.WorkerHost
	killCount := 0
	killAll := (targetWorker == "all")
	appCfg, err := OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	if killAll {
		fmt.Printf("Killing all %v workers\n", len(appCfg.Workers))
	}

	for _, rawWorkerHostname := range appCfg.Workers {
		currentWorker, err := worker.NewWorkerHostFromRaw(rawWorkerHostname)
		if err != nil {
			fmt.Println("Not able to powerdown worker")
			panic(1)
		}
		if killAll {
			killList = append(killList, currentWorker)
			continue
		}

		targetWorker = targetWorker + ":22"

		if currentWorker.Hostname != targetWorker {
			continue
		}
	}

	for _, currentWorker := range killList {
		fmt.Println("Powering off worker", currentWorker.Hostname)
		workerConn, _ := worker.NewConnectedWorker(currentWorker)
		cmd := command.NewPowerOffCommand()
		res, err := workerConn.SendCommand(cmd)
		err = nil
		if err != nil {
			fmt.Printf("    %s\n", err)
		} else {
			killCount++
			fmt.Printf("    %s\n", res.Data)
		}
		close(workerConn.Close)
	}

	fmt.Printf("Killed %v workers\n", killCount)

}

func pruneWorkers(cmd *cobra.Command, args []string) {
	fmt.Println("Prune not supported")
	// TODO: find the weakest worker and kill it

	// pruneTarget, err := strconv.Atoi(args[0])
	// if err != nil {
	// 	fmt.Println("Invalid prune value")
	// 	panic(1)
	// }
	// fmt.Println("Powering workers worker", targetWorker)

}

func main() {
	stats := &cobra.Command{
		Use:   "stats",
		Short: "Pulls stats from workers",
		Run:   getStats,
	}

	scaleCmd := &cobra.Command{
		Use:   "scale [command]",
		Short: "Scale Command",
	}

	scaleUp := &cobra.Command{
		Use:   "up [hostname]",
		Short: "Scales worker up",
		Args:  cobra.MinimumNArgs(1),
		Run:   scaleWorkerUp,
	}

	scaledown := &cobra.Command{
		Use:   "down [hostname]",
		Short: "Scales worker down",
		Args:  cobra.MinimumNArgs(1),
		Run:   scaleDownWorker,
	}

	kill := &cobra.Command{
		Use:   "kill [hostname]",
		Short: "Kills worker",
		Args:  cobra.MinimumNArgs(1),
		Run:   killWorker,
	}

	// TODO: should be able to restart worker or the whole cluster

	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(stats)
	rootCmd.AddCommand(kill)

	rootCmd.AddCommand(scaleCmd)
	scaleCmd.AddCommand(scaleUp)
	scaleCmd.AddCommand(scaledown)
	rootCmd.Execute()
}
