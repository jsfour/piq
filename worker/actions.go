package worker

import (
	"fmt"

	"github.com/jsmootiv/piq/command"
)

func KillWorkers(killList []WorkerHost) ([]WorkerHost, []WorkerHost) {
	var killedWorkers []WorkerHost
	var errorWorkers []WorkerHost
	for _, currentWorker := range killList {
		fmt.Println("Powering off worker", currentWorker.Hostname)
		workerConn, _ := NewConnectedWorker(currentWorker)
		cmd := command.NewPowerOffCommand()
		res, err := workerConn.SendCommand(cmd)
		err = nil
		if err != nil {
			errorWorkers = append(errorWorkers, currentWorker)
			fmt.Printf("    %s\n", err)
		} else {
			killedWorkers = append(killedWorkers, currentWorker)
			fmt.Printf("    %s\n", res.Data)
		}
		close(workerConn.Close)
	}
	return killedWorkers, errorWorkers
}
