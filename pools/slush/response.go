package slush

type SlushResponseWorkers struct {
	LastShare int    `json:"last_share"`
	Score     string `json:"score"`
	Alive     bool   `json:"alive"`
	Shares    int    `json:"shares"`
	Hashrate  int    `json:"hashrate"`
}

type SlushResponse struct {
	Username             string                 `json:"username"`
	UnconfirmedReward    string                 `json:"unconfirmed_reward"`
	Rating               string                 `json:"rating"`
	NmcSendThreshold     string                 `json:"nmc_send_threshold"`
	UnconfirmedNmcReward string                 `json:"unconfirmed_nmc_reward"`
	EstimatedReward      string                 `json:"estimated_reward"`
	Hashrate             string                 `json:"hashrate"`
	ConfirmedNmcReward   string                 `json:"confirmed_nmc_reward"`
	SendThreshold        string                 `json:"send_threshold"`
	ConfirmedReward      string                 `json:"confirmed_reward"`
	Workers              []SlushResponseWorkers `json:"workers"`
	Wallet               string                 `json:"wallet"`
}
