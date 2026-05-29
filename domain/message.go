package domain

type Message struct {
	ID       string `json:"id"`
	Data     any    `json:"data"`
	Function string `json:"function"`
	Done     bool   `json:"done"`
	Type     string `json:"type"`
	Code     string `json:"code"`
}
