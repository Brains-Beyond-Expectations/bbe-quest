package main

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/cmd"
	"github.com/Brains-Beyond-Expectations/bbe-quest/cli/misc/logger"
)

func main() {
	logger.Initialize()
	cmd.Execute()
}
