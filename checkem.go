package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

var root string
var board string
var schema [8]map[string]interface{}

func readJSON(filepath string) (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var res map[string]interface{}
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

//TODO: Parse schema
func readSchema() ([8]map[string]interface{}, error) {
	schemaFiles := [8]string{"agents_custom.json", "agents_standard.json", "offices_custom.json", "offices_standard.json", "openhouses_custom.json", "openhouses_standard.json", "properties_custom.json", "properties_standard.json"}

	res := [8]map[string]interface{}{}
	var err error
	schemaPref := root + "resources/es_mappings/es_"
	for i, name := range schemaFiles {
		res[i], err = readJSON(schemaPref + name)
		if err != nil {
			return res, err
		}
	}
	return res, nil
}

//gets FileInfo of all files under a directory
func getFilesInDir(folder string) ([]string, error) {
	dirEntries, err := ioutil.ReadDir(folder)
	if err != nil {
		return nil, err
	}
	res := []string{}
	for _, file := range dirEntries {
		if !file.IsDir() {
			res = append(res, folder+file.Name())
		}
	}
	return res, nil
}

func checkRoutine(jsonMap string, fin chan bool, log chan string) {
	//TODO: Read json mapping
	defer close(log)
	file, err := readJSON(jsonMap)
	if err != nil {
		fmt.Println(err)
		fin <- false
		log <- "FAIL"
	}
	//TODO: Read csv metadata
	//TODO: Check duplicate keys
	//TODO: Check valid nesting
	//TODO: Check keys missing or key not from metadata
	//TODO: Use channels to run checks on all mappings concurrently
	if file == nil {

	}
	fin <- true
	log <- "SUCC"
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: Checkem needs an argument!")
		os.Exit(1)
	}

	//set current run variables
	root = os.Getenv("HOME") + "/dev/ops/apps/runner/"
	board = os.Args[1]

	//check if environments/board.env exists
	_, err := ioutil.ReadFile(root + "environment/" + board + ".env")
	if err != nil {
		fmt.Println("ERROR:", board+".env", "not found!")
	}
	//check if queries exists
	_, err = ioutil.ReadFile(root + "queries/" + board + "/" + board + "_queries.json")
	if err != nil {
		fmt.Println("ERROR:", board+"_queries.json", "not found!")
	}
	_, err = ioutil.ReadFile(root + "queries/" + board + "/test_" + board + "_queries.json")
	if err != nil {
		fmt.Println("ERROR:", board+"_queries.json", "not found!")
	}

	//load common data
	//schema, acceptable data types
	//TODO: Load schema
	schema, err := readSchema()
	if err != nil {
		fmt.Println(err)
	}
	for i, schem := range schema {
		if i == 0 {

		}
		if schem == nil {

		}
		//fmt.Println(i)
		//fmt.Println(schem)
	}

	mappingsList, err := getFilesInDir(root + "mappings/" + board + "/")
	if err != nil {
		fmt.Println("ERROR: Unable to read mappings! Does the folder", board, "exists in mappings?")
		fmt.Println(err)
		os.Exit(1)
	}
	finChan := make(chan bool)
	logChans := make([]chan string, len(mappingsList))
	//spin off analysis of each mapping in their own goroutine
	for i, jsonMap := range mappingsList {
		fmt.Println(jsonMap)
		logChans[i] = make(chan string)
		go checkRoutine(jsonMap, finChan, logChans[i])
	}
	//wait for all to finish
	for range mappingsList {
		select {
		case test := <-finChan:
			fmt.Println(test)
		}
	}
}
