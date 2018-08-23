package worker

import (
	"fmt"

	"github.com/jsmootiv/piq/command"
	"github.com/jsmootiv/piq/util"
)

func executeBatchCommand(workerList []WorkerHost, cmd string) ([]WorkerHost, []WorkerHost) {
	var successfulWorkers []WorkerHost
	var errorWorkers []WorkerHost
	for _, currentWorker := range workerList {
		fmt.Printf("Running %v on worker %v\n", cmd, currentWorker.Hostname)
		workerConn, _ := NewConnectedWorker(currentWorker)
		res, err := workerConn.SendCommand(cmd)
		err = nil
		if err != nil {
			errorWorkers = append(errorWorkers, currentWorker)
			fmt.Printf("    %s\n", err)
		} else {
			successfulWorkers = append(successfulWorkers, currentWorker)
			fmt.Printf("    %s\n", res.Data)
		}
		close(workerConn.Close)
	}
	return successfulWorkers, errorWorkers
}

type ExecFunc = func(killList []WorkerHost) ([]WorkerHost, []WorkerHost)

func KillWorkers(killList []WorkerHost) ([]WorkerHost, []WorkerHost) {
	fmt.Printf("Killing %v workers\n", len(killList))
	cmd := command.NewPowerOffCommand()
	return executeBatchCommand(killList, cmd)
}

func RebootWorkers(rebootList []WorkerHost) ([]WorkerHost, []WorkerHost) {
	fmt.Printf("Rebooting %v workers\n", len(rebootList))
	cmd := command.NewRebootCommand()
	return executeBatchCommand(rebootList, cmd)
}

func RunExecutor(targetWorker string, executorFunc ExecFunc) ([]WorkerHost, []WorkerHost) {
	var targetList []WorkerHost

	appCfg, err := util.OpenConfig("./config.json")
	if err != nil {
		fmt.Println("Failed to load config: ", err)
		panic(1)
	}

	if targetWorker == "all" {
		for _, rawWorkerHostname := range appCfg.Workers {
			currentWorker, err := NewWorkerHostFromRaw(rawWorkerHostname)
			if err != nil {
				fmt.Println(err)
				panic(1)
			}
			targetList = append(targetList, currentWorker)
		}
	} else {
		targetWorker := targetWorker + ":22"
		for _, rawWorkerHostname := range appCfg.Workers {
			currentWorker, err := NewWorkerHostFromRaw(rawWorkerHostname)
			if err != nil {
				fmt.Println("Not able to generate name")
				panic(1)
			}

			if currentWorker.Hostname == targetWorker {
				targetList = append(targetList, currentWorker)
				break
			}
		}
	}
	return executorFunc(targetList)
}
