package engine

import "fmt"

type Refusal struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Received any    `json:"received,omitempty"`
	Expected any    `json:"expected,omitempty"`
}

func (r *Refusal) Error() string {
	return fmt.Sprintf("%s: %s (received %v, expected %v)", r.Code, r.Message, r.Received, r.Expected)
}
