package types

type Daemon struct {
	Key    string `json:"key"`
	Cmd    string `json:"cmd"`
	Dir    string `json:"dir,omitempty"`
	Active bool   `json:"active"`
	Status string `json:"status"`
}
