package booze

import "encoding/json"

type Payload struct {
	ID     string          `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

func (p *Payload) Unmarshal(v interface{}) error {
	return json.Unmarshal(p.Params, v)
}

func (p *Payload) Marshal(v interface{}) (err error) {
	p.Params, err = json.Marshal(v)
	return err
}
