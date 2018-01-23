package types

type Daemon struct {
	Key     string `json:"key"`
	Cmd     string `json:"cmd"`
	Dir     string `json:"dir,omitempty"`
	Running bool   `json:"running"`
}
