package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/jsmootiv/piq/command"
	"github.com/jsmootiv/piq/pools"
	"github.com/jsmootiv/piq/util"
	"github.com/jsmootiv/piq/worker"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

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

func printPoolStats(poolFeed chan pools.Pool) {
	var tableData [][]string
	minPayout := 0.01 // TODO: move this into config
	fmt.Println("Getting pool stats")
	// TODO: fill in error
	for myRes := range poolFeed {
		rewardFl, _ := strconv.ParseFloat(myRes.Reward, 64)
		prog := rewardFl / minPayout
		row := []string{
			myRes.Name,
			myRes.Reward,
			fmt.Sprintf("%.2f", prog*100),
			myRes.Hashrate,
		}
		tableData = append(tableData, row)
	}
	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Name", "Reward", "Payout Progress", "Hashrate"})
	for _, v := range tableData {
		table.Append(v)
	}
	table.Render() // Send output
}

func getWorkerStats(workers []string) (chan command.CommandResponse, chan worker.WorkerHost) {
	fmt.Println("Starting worker stats collection")
	var hostWg sync.WaitGroup
	var cmdWg sync.WaitGroup
	workerHosts, err := worker.NewWorkerHostBatch(workers)
	responseFeed := make(chan command.CommandResponse, len(workerHosts))
	runningWorkers := make(chan *worker.WorkerConnection, len(workerHosts))
	failedWorkers := make(chan worker.WorkerHost, len(workerHosts))
	go func() {
		defer close(responseFeed)
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
				close(conn.Close)
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
	}()
	return responseFeed, failedWorkers
}

func getStats(statsCmd *cobra.Command, args []string) {
	statArg := strings.ToLower(args[0])
	appCfg, err := util.OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	// TODO: paralelize this operation
	if statArg == "all" || statArg == "worker" || statArg == "workers" {
		responseFeed, failedWorkers := getWorkerStats(appCfg.Workers)
		printDone := printStatsWorker(responseFeed, failedWorkers)
		<-printDone
	}
	if statArg == "all" || statArg == "pool" || statArg == "pools" {
		poolsFeed, _ := pools.GetPools(appCfg.Pools)
		printPoolStats(poolsFeed)
	}
}

func printWorkers(workers []worker.WorkerHost) {
	for _, wk := range workers {
		fmt.Println("   ", wk.Hostname)
	}
}

func rebootWorkers(cmd *cobra.Command, args []string) {
	targetWorker := strings.ToLower(args[0])
	successKills, errKills := worker.RunExecutor(targetWorker, worker.RebootWorkers)
	fmt.Printf("Rebooted %v workers\n", len(successKills))
	if len(errKills) > 0 {
		fmt.Printf("Issue rebooting %v workers\n", len(errKills))
		printWorkers(errKills)
	}
}

func killWorker(cmd *cobra.Command, args []string) {
	targetWorker := strings.ToLower(args[0])
	successKills, errKills := worker.RunExecutor(targetWorker, worker.KillWorkers)
	fmt.Printf("Killed %v workers\n", len(successKills))
	if len(errKills) > 0 {
		fmt.Printf("Issue killing %v workers\n", len(errKills))
		printWorkers(errKills)
	}
}

func pruneWorkers(cmd *cobra.Command, args []string) {
	fmt.Println("Prune not supported yet")
	// TODO: find the weakest worker and kill it
}

func main() {
	stats := &cobra.Command{
		Use:   "stats",
		Short: "Pulls stats from worker or pool",
		Args:  cobra.MinimumNArgs(1),
		Run:   getStats,
	}

	kill := &cobra.Command{
		Use:   "kill [hostname]",
		Short: "Kills worker",
		Args:  cobra.MinimumNArgs(1),
		Run:   killWorker,
	}

	reboot := &cobra.Command{
		Use:   "reboot [hostname]",
		Short: "reboots worker",
		Args:  cobra.MinimumNArgs(1),
		Run:   rebootWorkers,
	}

	rootCmd := &cobra.Command{Use: "app"}
	rootCmd.AddCommand(stats)
	rootCmd.AddCommand(kill)
	rootCmd.AddCommand(reboot)

	rootCmd.Execute()
}
