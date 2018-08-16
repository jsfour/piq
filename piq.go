package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jsmootiv/piq/command"
	"github.com/jsmootiv/piq/worker"
	"github.com/olekukonko/tablewriter"
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

func formatHashrateString(hRate float64) string {
	return fmt.Sprintf("%.4f", hRate/1e3)
}

func printStatsWorker(responseFeed chan command.CommandResponse, downFeed chan worker.WorkerHost) chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		var summaryResponses []command.SummaryRes
		var tableData [][]string
		var totalAvgHashrate float64
		var totalHardwareErrors int

		for rawRes := range responseFeed {
			var myRes command.SummaryRes
			err := json.Unmarshal(rawRes.Data, &myRes)
			if err != nil {
				fmt.Println(rawRes.Source, "Error parsing", err)
				continue
			}
			myRes.Source = rawRes.Source
			if myRes.Error != "" {
				fmt.Println(myRes.Source, "Error", myRes.Error)
				continue
			}
			summaryResponses = append(summaryResponses, myRes)
		}

		sort.Sort(sort.Reverse(command.ByHashrate(summaryResponses)))

		for _, myRes := range summaryResponses {
			totalAvgHashrate += myRes.Summary[0].GhsAv
			totalHardwareErrors += myRes.Summary[0].HardwareErrors
			row := []string{
				"Up",
				myRes.Source,
				formatHashrateString(myRes.Summary[0].GhsAv),
				myRes.Summary[0].Ghs5s,
				strconv.Itoa(myRes.Summary[0].HardwareErrors),
			}
			tableData = append(tableData, row)
		}

		for wkr := range downFeed {
			row := []string{
				"Down",
				wkr.Hostname,
				"0",
				"0",
				"0",
			}
			tableData = append(tableData, row)
		}

		table := tablewriter.NewWriter(os.Stdout)
		table.SetHeader([]string{"Status", "Worker", "Avg Hashrate (th/s)", "5sec Hashrate", "Hardware Errs"})
		table.SetFooter([]string{"Total", "", formatHashrateString(totalAvgHashrate), "", ""})

		for _, v := range tableData {
			table.Append(v)
		}
		table.Render() // Send output
	}()
	return done
}

func getStats(cmd *cobra.Command, args []string) {
	fmt.Println("Starting stats collection")
	var hostWg sync.WaitGroup
	var cmdWg sync.WaitGroup

	appCfg, err := OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	workerHosts, err := worker.NewWorkerHostBatch(appCfg.Workers)
	runningWorkers := make(chan *worker.WorkerConnection, len(workerHosts))
	failedWorkers := make(chan worker.WorkerHost, len(workerHosts))
	responseFeed := make(chan command.CommandResponse, len(workerHosts))
	printDone := printStatsWorker(responseFeed, failedWorkers)
	if err != nil {
		fmt.Println("Failed to read workers: ", err)
		panic(1)
	}

	for _, wkrHost := range workerHosts {
		hostWg.Add(1)
		go func(wkrHost worker.WorkerHost) {
			defer hostWg.Done()
			workerConn, err := worker.NewConnectedWorker(wkrHost)

			if err != nil {
				failedWorkers <- wkrHost
				return
			}
			runningWorkers <- workerConn
		}(wkrHost)
	}

	hostWg.Wait()
	close(runningWorkers)
	close(failedWorkers)
	fmt.Printf("%v workers up, %v workers down\n", len(runningWorkers), len(failedWorkers))
	for conn := range runningWorkers {
		cmdWg.Add(1)
		go func(conn *worker.WorkerConnection) {
			defer worker.Stop()
			cmd := command.NewSummaryCommand()
			res, err := conn.SendCommand(cmd)
			if err != nil {
				res.Data = command.NewErrorJson(err)
			}
			responseFeed <- *res
			cmdWg.Done()
		}(conn)
	}
	cmdWg.Wait()
	close(responseFeed)
	<-printDone
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
	appCfg, err := OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	if targetWorker == "all" {
		fmt.Printf("Killing all %v workers\n", len(appCfg.Workers))
		for _, rawWorkerHostname := range appCfg.Workers {
			currentWorker, err := worker.NewWorkerHostFromRaw(rawWorkerHostname)
			if err != nil {
				fmt.Println(err)
				panic(1)
			}
			killList = append(killList, currentWorker)
		}
	} else {
		targetWorker := targetWorker + ":22"
		for _, rawWorkerHostname := range appCfg.Workers {
			currentWorker, err := worker.NewWorkerHostFromRaw(rawWorkerHostname)
			if err != nil {
				fmt.Println("Not able to powerdown worker")
				panic(1)
			}

			if currentWorker.Hostname == targetWorker {
				killList = append(killList, currentWorker)
				break
			}
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
