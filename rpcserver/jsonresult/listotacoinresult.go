package jsonresult

type ListOTACoinIdx struct {
	FromHeight uint64   `json:"FromHeight"`
	ToHeight   uint64   `json:"ToHeight"`
	Indexs     []string `json:"Indexs"`
}
