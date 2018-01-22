package types

type Daemon struct {
	Key     string `json:"key"`
	Cmd     string `json:"cmd"`
	Running bool   `json:"running"`
}
