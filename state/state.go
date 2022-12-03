package state

import (
	_ "embed"
	"void/types"
)

//go:embed services.yaml
var (
	servicesByte []byte
	services     types.Document
)

func init() {

}
