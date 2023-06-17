package main

import (
	"fmt"
	"os"

	"github.com/julien-sobczak/the-notewriter/internal/core"
)

func CheckConfig() {
	err := core.CurrentConfig().Check()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
