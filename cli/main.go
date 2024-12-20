package main

import (
	"github.com/Brains-Beyond-Expectations/bbe-quest/cmd"
	"github.com/Brains-Beyond-Expectations/bbe-quest/services/logger"
)

func main() {
	logger.Initialize()
	cmd.Execute()
}
