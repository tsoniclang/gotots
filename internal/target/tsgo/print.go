package tsgo

import "encoding/base64"

type PrintOptions struct {
	PreserveSourceNewlines        bool
	NeverASCIIEscape              bool
	TerminateUnterminatedLiterals bool
}

type printNodeParams struct {
	Data                          string `json:"data"`
	PreserveSourceNewlines        bool   `json:"preserveSourceNewlines,omitempty"`
	NeverASCIIEscape              bool   `json:"neverAsciiEscape,omitempty"`
	TerminateUnterminatedLiterals bool   `json:"terminateUnterminatedLiterals,omitempty"`
}

func (c *Client) PrintNode(node Node, options PrintOptions) (string, error) {
	data, err := EncodeNode(node)
	if err != nil {
		return "", err
	}
	return c.print(printNodeParams{
		Data:                          base64.StdEncoding.EncodeToString(data),
		PreserveSourceNewlines:        options.PreserveSourceNewlines,
		NeverASCIIEscape:              options.NeverASCIIEscape,
		TerminateUnterminatedLiterals: options.TerminateUnterminatedLiterals,
	})
}
