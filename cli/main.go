package main

import (
	"github.com/nicolajv/bbe-quest/cmd"
	"github.com/nicolajv/bbe-quest/services/logger"
)

func main() {
	logger.Initialize()
	cmd.Execute()
}
