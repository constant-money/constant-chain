package debugtool

import (
	"fmt"
	"testing"
)

func TestGenTestAccount(t *testing.T) {
	GenTestAccount("", 100)
	fmt.Println(len(TestAccounts))
	tool :=  new(DebugTool).InitDevNet()
	SendPrv2TestAccounts(tool, 10)
}
