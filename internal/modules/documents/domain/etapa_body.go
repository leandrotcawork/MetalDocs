package domain

import "encoding/json"

type EtapaBody struct {
	Blocks []json.RawMessage
}
