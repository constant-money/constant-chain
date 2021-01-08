package debugtool

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
)

type DebugTool struct {
	url string
}

func (this *DebugTool) InitMainnet() *DebugTool {
	if this == nil {
		this = new(DebugTool)
	}
	this.url = "https://mainnet.incognito.org/fullnode"
	return this
}

func (this *DebugTool) InitTestnet() *DebugTool {
	if this == nil {
		this = new(DebugTool)
	}
	this.url = "http://51.83.36.184:20002"
	return this
}

func (this *DebugTool) InitLocal(port string) *DebugTool {
	if this == nil {
		this = new(DebugTool)
	}
	this.url = "http://127.0.0.1:" + port
	return this
}

func (this *DebugTool) InitDevNet() *DebugTool {
	if this == nil {
		this = new(DebugTool)
	}
	this.url = "http://139.162.55.124:8334"
	return this
}

func (this *DebugTool) SendPostRequestWithQuery(query string) ([]byte, error) {
	if len(this.url) == 0 {
		return []byte{}, errors.New("Debugtool has not set mainnet or testnet")
	}
	var jsonStr = []byte(query)
	req, _ := http.NewRequest("POST", this.url, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return []byte{}, err
	} else {
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return []byte{}, err
		}
		return body, nil
	}
}
