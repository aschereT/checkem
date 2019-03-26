package main

import (
	"os"
	"fmt"
	"io/ioutil"
)

const root = "~/dev/ops/apps/runner/"

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: Checkem needs an argument!")
		os.Exit(1)
	}

	board := os.Args[1]
	fmt.Println("Checking", board)

	mappingsList, err := ioutil.ReadDir(root + "mappings/" + board + "/")
	if err != nil {
		fmt.Println("ERROR: Unable to read mappings! Does the folder", board, "exists in mappings?")
		os.Exit(1)
	}
	

}