// Copyright (c) 2014 Square, Inc
// +build linux darwin

package main

import (
	"encoding/json"
	"fmt"

	"inspect/cpustat/osmain"
)

func main() {

	processState := osmain.GetProcessStat()

	b, _ := json.Marshal(processState)
	fmt.Println("processStateChan---", string(b))

}
