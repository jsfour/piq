package pools

import (
	"fmt"

	"github.com/jsmootiv/piq/pools/slush"
)

type Pool struct {
	Name      string `json:"name"`
	AccountId string `json:"accountId"`
	Reward    string `json:"reward"`
	Hashrate  string `json:"hashrate"`
}

func GetPools(poolParams []Pool) (chan Pool, error) {
	resPools := make(chan Pool, len(poolParams))
	go func() {
		defer close(resPools)
		for _, p := range poolParams {
			if p.Name == "slushpool" {
				res, _ := slush.GetSlush(p.AccountId)
				nuPool := Pool{
					Name:      p.Name,
					AccountId: p.AccountId,
					Reward:    res.ConfirmedReward,
					Hashrate:  res.Hashrate,
				}
				resPools <- nuPool
			} else {
				fmt.Println("Pool not found", p.Name)
			}
		}
	}()
	return resPools, nil
}
