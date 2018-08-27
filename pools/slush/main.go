package slush

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const slushEndpoint = "https://slushpool.com"

func GetSlush(accountNumber string) (SlushResponse, error) {
	var res SlushResponse
	url := fmt.Sprintf("%s/accounts/profile/json/%s", slushEndpoint, accountNumber)
	// "https://slushpool.com/accounts/profile/json/accountNumber"
	httpRes, err := http.Get(url)

	// TODO: finish this
	if err != nil {
		return res, err
	}

	if httpRes.StatusCode != 200 {
		fmt.Println("There was an error getting slushpool", url)
	}

	bodyBytes, err := ioutil.ReadAll(httpRes.Body)
	if err != nil {
		return res, err
	}

	err = json.Unmarshal(bodyBytes, &res)
	if err != nil {
		return res, err
	}

	return res, err
}
