package command

import "fmt"

type ResponseStatus int

const (
	Failed  ResponseStatus = iota
	Success ResponseStatus = iota
)

type StatusResItem struct {
	// TODO: move me
	Status      string `json:"STATUS"`
	When        int64  `json:"When"`
	Code        int64  `json:"Code"`
	Msg         string `json:"Msg"`
	Description string `json:"Description"`
}

type SummaryResItem struct {
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

type SummaryRes struct {
	Status  []StatusResItem  `json:"STATUS"`
	Summary []SummaryResItem `json:"SUMMARY"`
	Error   string           `json:"error";omitempty`
	Source  string
}

type StatsRes struct {
	Status []StatusResItem          `json:"STATUS"`
	Stats  []map[string]interface{} `json:"STATS"`
}

type CommandResponse struct {
	Data   []byte
	Source string
	Status ResponseStatus
}

func NewErrorJson(err error) []byte {
	str := fmt.Sprint(`{ "error": "`, err.Error(), `"}`)
	return []byte(str)
}

func NewCommandResponse(source string) *CommandResponse {
	return &CommandResponse{
		Data:   []byte{},
		Source: source,
	}
}

func NewSummaryCommand() string {
	return `echo '{"command": "summary"}' | nc localhost 4028`
}

func NewRestartCommand() string {
	return "/sbin/reboot"
}

func NewStatsCommand() string {
	return `echo '{"command": "stats"}' | nc localhost 4028`
}

func NewScaleUpCommand(poolIndex int) string {
	// TODO: probably should figure out the best command here
	return fmt.Sprintf(`echo '{"command": "enablepool", "parameter": "%v"}' | nc localhost 4028`, poolIndex)
}

func NewPowerOffCommand() string {
	return "/sbin/poweroff"
}

func NewScaleDownCommand(poolIndex int) string {
	// Turns out disablepool doesnt do what I need
	return fmt.Sprintf(`echo '{"command": "disablepool", "parameter": "%v"}' | nc localhost 4028`, poolIndex)
}
