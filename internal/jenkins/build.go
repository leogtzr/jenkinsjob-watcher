package jenkins

import "fmt"

type Build struct {
	Building        bool    `json:"building"`
	Description     string  `json:"description"`
	DisplayName     string  `json:"displayName"`
	FullDisplayName string  `json:"fullDisplayName"`
	ID              string  `json:"id"`
	Number          int     `json:"number"`
	Result          *string `json:"result"`
	URL             string  `json:"url"`
	InProgress      bool    `json:"inProgress"`
}

func (b Build) String() string {
	result := "null"
	if b.Result != nil {
		result = *b.Result
	}

	return fmt.Sprintf(
		"Build #%d (%s) | building=%t | result=%s",
		b.Number,
		b.DisplayName,
		b.Building,
		result,
	)
}
