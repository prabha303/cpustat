// Copyright (c) 2014 Square, Inc
// +build linux darwin

package main

import (
	"encoding/json"
	"fmt"
	"strconv"

	"inspect/cpustat/osmain"
)

func main() {
	currentPID := strconv.Itoa(10907)
	processState := osmain.GetProcessStat(currentPID)

	b, _ := json.Marshal(processState)
	fmt.Println("processStateChan---", string(b))

}
