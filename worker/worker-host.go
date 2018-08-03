package worker

import (
	"strings"
)

type WorkerHost struct {
	Username string
	Password string
	Hostname string
}

func NewWorkerHostFromRaw(rawHostname string) (WorkerHost, error) {
	addySlit := strings.Split(rawHostname, "@")
	password := addySlit[0]
	hostname := addySlit[1] + ":22"
	out := WorkerHost{
		Username: "root",
		Password: password,
		Hostname: hostname,
	}

	return out, nil
}

func NewWorkerHostBatch(rawWorkers []string) ([]WorkerHost, error) {
	var workers []WorkerHost

	for _, myRaw := range rawWorkers {
		newHost, err := NewWorkerHostFromRaw(myRaw)
		if err != nil {
			return workers, err
		}
		workers = append(workers, newHost)
	}
	return workers, nil
}
