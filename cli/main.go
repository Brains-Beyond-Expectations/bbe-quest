package main

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cmd"
	"github.com/Brains-Beyond-Expectations/bbe-quest/misc/logger"
)

func main() { // coverage-ignore
	logger.Initialize()
	cmd.Execute()
}
