package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
	"encoding/csv"
)

type empty struct{}

type schemaStandard struct {
	SchemaMappings map[string]map[string]interface{} `json:"mappings"`
}

//schemaNest is the struct of choice for customs. If nested = true, fields should be mapped to one of the properties
type schemaNest struct {
	nested     bool
	properties map[string]empty
}

var root string
var board string
var standardSchemas map[string]map[string]schemaNest

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

func readCSV(filepath string) (map[string]bool, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	res := make(map[string]bool)
	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	for _, fieldname := range records {
		res[fieldname[0]] = false
	}
	delete(res, "SystemName")
	return res, nil
}

func readSchema(filepath string) (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var res schemaStandard
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res.SchemaMappings["_doc"]["properties"].(map[string]interface{}), nil
}

//loads all schemas to memory
func readSchemas() error {
	//TODO: Just use the resource name
	resources := map[string]string{"agents": "agent", "offices": "office", "openhouses": "openhouse", "properties": "property"}
	schemaPref := root + "resources/es_mappings/es_"

	standardSchemas = map[string]map[string]schemaNest{}

	fin := make(chan error, 4)
	for resourceType := range resources {
		go func(resType string) {
			propertiesMap, err := readSchema(schemaPref + resType + "_standard.json")
			curSchem := map[string]schemaNest{}
			if err != nil {
				fin <- err
				return
			}
			for k := range propertiesMap {
				curNesting := schemaNest{nested: propertiesMap[k].(map[string]interface{})["type"] == "nested", properties: map[string]empty{}}
				if curNesting.nested {
					for p := range propertiesMap[k].(map[string]interface{})["properties"].(map[string]interface{}) {
						curNesting.properties[p] = empty{}
					}
				}
				curSchem[k] = curNesting
			}
			standardSchemas[resources[resType]] = curSchem
			fin <- nil
		}(resourceType)
	}

	for i := 0; i < 4; i++ {
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
			res = append(res, file.Name())
		}
	}
	return res, nil
}

//given a mapping json filename, returns resource and class
func filenameChunker(filename string) (string, string) {
	res := strings.Split(filename, "_")
	return res[1], strings.Join(res[2:len(res)-1], "_")
}

func clamp(val int, lo int, hi int) int {
	if val > hi {
		return hi
	}
	if val < lo {
		return lo
	}
	return val
}

//https://stackoverflow.com/questions/18342784/how-to-iterate-through-a-map-in-golang-in-order
func sortKeys(m map[string]interface{}) []string {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	return keys
}

func checkRoutine(jsonMap string, fin chan int, log *strings.Builder) {
	fmt.Fprintln(log, jsonMap)
	mapping, err := readJSON(root + "mappings/" + board + "/" + jsonMap)
	if err != nil {
		fmt.Fprintln(log, err)
		fin <- 1
		return
	}
	resource, class := filenameChunker(jsonMap)
	csvList, err := readCSV(root + "metadata/" + board + "_" + resource + "_" + class + ".csv")
	resource = strings.TrimPrefix(resource, "active")
	if err != nil {
		fmt.Fprintln(log, err)
		fin <- 1
		return
	}

	//TODO: wrap strings builder, so no need to manually keep track
	errCount := 0
	//stores all non-nested values we have encountered so far
	mappedFieldvals := map[string]string{}
	//TODO: Read csv metadata
	if mapping != nil {
		detKeys := sortKeys(mapping)
		for _, key := range detKeys {
			switch mapping[key].(type) {
			case string:
				mappedVal := mapping[key].(string)
				//check if in metadata
				_, ex := csvList[key]
				if !ex {
					fmt.Fprintln(log, "	", key+":", "not in metadata")
					errCount++
				} else {
					csvList[key] = true
				}
				if mappedVal == "" {
					//ignore empty fields
					continue
				}
				//Check if in schema
				aNest, ex := standardSchemas[resource][mappedVal]
				if !ex {
					fmt.Fprintln(log, "	", key+":", mappedVal, "is not in", resource+"'s", "standard schema")
					errCount++
				} else if aNest.nested {
					fmt.Fprintln(log, "	", key+":", "is supposed to be a nest but was mapped to", mappedVal)
					errCount++
				}
				//Check if another field is already mapped to the same thing
				other, ex := mappedFieldvals[mappedVal]
				if ex {
					fmt.Fprintln(log, "	", key+":", mappedVal, "is repeated with", other)
					errCount++
				} else {
					mappedFieldvals[mappedVal] = key
				}
			case interface{}:
				//Nested
				assertedNest := mapping[key].([]interface{})
				nestSchem, ok := assertedNest[0].(string)
				if !ok {
					fmt.Fprintln(log, "	", key+":", "Nesting", key, "is missing the custom type")
					errCount++
					continue
				}
				nesting, ok := assertedNest[1].(map[string]interface{})
				if !ok {
					fmt.Fprintln(log, "	", key+":", "Nesting", key, "is missing mappings")
					errCount++
					continue
				}
				nestKeyinside, nestName, nestType := false, false, false
				aNest, ex := standardSchemas[resource][nestSchem]
				if !ex || !aNest.nested {
					fmt.Fprintln(log, "	", key+":", "Nesting", key, "has an invalid nesting", nestSchem)
					errCount++
				}
				for mapField := range nesting {
					switch mapField {
					case "Name":
						nestName = true
					case "Type":
						nestType = true
						_, ok := nesting[mapField].(string)
						if !ok {
							fmt.Fprintln(log, "	", key+":", "Nesting", key, "has empty Type")
							errCount++
						}
					default:
						//check if in metadata
						_, ex = csvList[mapField]
						if !ex {
							fmt.Fprintln(log, "	", key+":", "not in metadata")
							errCount++
						} else {
							csvList[mapField] = true
						}
						if mapField == key {
							nestKeyinside = true
						}
						_, ex = standardSchemas[resource][nestSchem].properties[nesting[mapField].(string)]
						if !ex {
							fmt.Fprintln(log, "	", key+":", "Nested property", mapField, "has an invalid nesting", nesting[mapField].(string), "for", nestSchem)
							errCount++
						}
					}
				}
				if !nestName {
					fmt.Fprintln(log, "	", key+":", "Missing Name inside nesting")
					errCount++
				}
				if !nestType {
					fmt.Fprintln(log, "	", key+":", "Missing Type inside nesting")
					errCount++
				}
				if !nestKeyinside {
					fmt.Fprintln(log, "	", key+":", "Missing itself inside nesting")
					errCount++
				}
			default:
				fmt.Fprintln(log, "	", key+":", "Unknown mapping")
				errCount++
			}
		}
	} else {
		fmt.Fprintln(log, "Mapping is nil!")
	}
	//TODO: Check keys missing or key not from metadata
	for index, csvRec := range csvList {
		if !csvRec {
			fmt.Fprintln(log, "	", index+":", "Not found in mappings")
			errCount++
		}
	}
	fin <- errCount
	return
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: checkem <board name>")
		os.Exit(1)
	}

	//set current run variables
	root = os.Getenv("HOME") + "/dev/ops/apps/runner/"
	board = os.Args[1]

	//check if environments/board.env exists
	_, err := ioutil.ReadFile(root + "environment/" + board + ".env")
	if err != nil {
		fmt.Println(err)
	}
	//check if queries exists
	_, err = ioutil.ReadFile(root + "queries/" + board + "/" + board + "_queries.json")
	if err != nil {
		fmt.Println(err)
	}
	_, err = ioutil.ReadFile(root + "queries/" + board + "/test_" + board + "_queries.json")
	if err != nil {
		fmt.Println(err)
	}

	//load common data
	//schema, acceptable data types
	err = readSchemas()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mappingsList, err := getFilesInDir(root + "mappings/" + board + "/")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	finChan := make(chan int, len(mappingsList))
	loggers := make([]*strings.Builder, len(mappingsList))
	//spin off analysis of each mapping in their own goroutine
	for i, jsonMap := range mappingsList {
		loggers[i] = new(strings.Builder)
		go checkRoutine(jsonMap, finChan, loggers[i])
	}

	errorSum := 0
	//wait for all to finish
	for range mappingsList {
		select {
		case errorCount := <-finChan:
			errorSum += errorCount
		}
	}

	//Print out all logs in order
	for _, log := range loggers {
		fmt.Print(log.String())
	}
	fmt.Println("Count", errorSum)
	os.Exit(clamp(errorSum, 0, 255))
}
