package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type empty struct{}

//SchemaStandard is the standard schema's struct, discarding uneeded fields
type SchemaStandard struct {
	//Settings       interface{}                       `json:"settings"`
	SchemaMappings map[string]map[string]interface{} `json:"mappings"`
}

//SchemaCustom is the custom schema's struct. Simple.
type SchemaCustom struct {
	SchemaMappings map[string]interface{} `json:"properties"`
}

var root string
var board string
var standardSchemas [4]map[string]empty
var customSchemas [4]map[string]bool

//used for checkRoutines
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

//like readJSON, but reads standard schema JSON only
func readSchemaStandard(filepath string) (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var res SchemaStandard
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res.SchemaMappings["_doc"]["properties"].(map[string]interface{}), nil
}

//like readJSON, but reads standard schema JSON only
func readSchemaCustom(filepath string) (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var res SchemaCustom
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res.SchemaMappings, nil
}

//TODO: Parse schema
func readSchemas() error {
	//TODO: Just use the resource name
	standardFiles := [4]string{"agents_standard.json", "offices_standard.json", "openhouses_standard.json", "properties_standard.json"}
	customFiles := [4]string{"agents_custom.json", "offices_custom.json", "openhouses_custom.json", "properties_custom.json"}
	schemaPref := root + "resources/es_mappings/es_"

	fin := make(chan error, 2)
	//process standard schema
	go func() {
		for i, filename := range standardFiles {
			propertiesMap, err := readSchemaStandard(schemaPref + filename)
			curSchem := map[string]empty{}
			if err != nil {
				fin <- err
				return
			}
			for k := range propertiesMap {
				curSchem[k] = empty{}
			}
			standardSchemas[i] = curSchem
		}
		fin <- nil
	}()

	go func() {
		for i, filename := range customFiles {
			propertiesMap, err := readSchemaCustom(schemaPref + filename)
			curSchem := map[string]bool{}
			if err != nil {
				fin <- err
				return
			}
			for k := range propertiesMap {
				//TODO: Set true if nesting, and false otherwise
				if propertiesMap[k].(map[string]interface{})["type"] == "nested" {
					curSchem[k] = true
				} else {
					curSchem[k] = false
				}
				//_, ex := propertiesMap[k]["type"]
			}
			customSchemas[i] = curSchem
		}
		fin <- nil
	}()

	for i := 0; i < 2; i++ {
		select {
		case err := <-fin:
			if err != nil {
				return err
			}
		}
	}
	return nil
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
		return
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
	err = readSchemas()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mappingsList, err := getFilesInDir(root + "mappings/" + board + "/")
	if err != nil {
		fmt.Println("ERROR: Unable to read mappings! Does the folder", board, "exists in mappings?")
		fmt.Println(err)
		os.Exit(1)
	}
	finChan := make(chan bool, len(mappingsList))
	logChans := make([]chan string, len(mappingsList))
	//spin off analysis of each mapping in their own goroutine
	for i, jsonMap := range mappingsList {
		logChans[i] = make(chan string)
		go checkRoutine(jsonMap, finChan, logChans[i])
	}
	//wait for all to finish
	for range mappingsList {
		select {
		case <-finChan:
		}
	}
	//TODO: Print off logs in a deterministic manner
}
