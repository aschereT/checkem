package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

//TODO: get home directory
const root = "/home/aschere/dev/ops/apps/runner/"

func ReadJSON(filepath string) map[string]interface{} {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("ERROR: Can't open JSON file", filepath)
		fmt.Println(err)
		return nil
	}
	var res map[string]interface{}
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		fmt.Println("ERROR: Can't parse JSON file", filepath)
		fmt.Println(err)
		return nil
	}
	return res
}

func ReadSchema() [8]map[string]interface{} {
	res := [8]map[string]interface{}{}
	const schemaPref = root + "resources/es_mappings/es_"
	res[0] = ReadJSON(schemaPref + "agents_custom.json")
	res[1] = ReadJSON(schemaPref + "agents_standard.json")
	res[2] = ReadJSON(schemaPref + "offices_custom.json")
	res[3] = ReadJSON(schemaPref + "offices_standard.json")
	res[4] = ReadJSON(schemaPref + "openhouses_custom.json")
	res[5] = ReadJSON(schemaPref + "openhouses_standard.json")
	res[6] = ReadJSON(schemaPref + "properties_custom.json")
	res[7] = ReadJSON(schemaPref + "properties_standard.json")
	return res
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("ERROR: Checkem needs an argument!")
		os.Exit(1)
	}

	board := os.Args[1]
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
	//check if mappings/board exists
	mappingsList, err := ioutil.ReadDir(root + "mappings/" + board + "/")
	if err != nil {
		fmt.Println("ERROR: Unable to read mappings! Does the folder", board, "exists in mappings?")
		fmt.Println(err)
		os.Exit(1)
	}

	//load common data
	//schema, acceptable data types
	schema := ReadSchema()
	for i, schem := range schema {
		fmt.Println(i)
		fmt.Println(schem)
	}

	for _, jsonMap := range mappingsList {
		fmt.Println(jsonMap.Name())
	}

	return
}
