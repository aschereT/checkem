package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strings"
)

type empty struct{}

//schemaStandard is the standard schema's struct, discarding uneeded fields
type schemaStandard struct {
	//Settings       interface{}                       `json:"settings"`
	SchemaMappings map[string]map[string]interface{} `json:"mappings"`
}

//schemaCustom is the custom schema's struct. Simple.
type schemaCustom struct {
	SchemaMappings map[string]interface{} `json:"properties"`
}

//schemaNest is the struct of choice for customs. If nested = true, fields should be mapped to one of the properties
type schemaNest struct {
	nested     bool
	properties map[string]empty
}

var root string
var board string
var standardSchemas map[string]map[string]empty
var customSchemas map[string]map[string]schemaNest
var adt map[string][]string

//used for reading ADT
func readAdt() (map[string][]string, error) {
	file, err := ioutil.ReadFile("adt.json")
	if err != nil {
		return nil, err
	}
	var res map[string][]string
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res, nil
}

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
	var res schemaStandard
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res.SchemaMappings["_doc"]["properties"].(map[string]interface{}), nil
}

//like readJSON, but reads custom schema JSON only
func readSchemaCustom(filepath string) (map[string]interface{}, error) {
	file, err := ioutil.ReadFile(filepath)
	if err != nil {
		return nil, err
	}
	var res schemaCustom
	err = json.Unmarshal([]byte(file), &res)
	if err != nil {
		return nil, err
	}
	return res.SchemaMappings, nil
}

//loads all schemas to memory
func readSchemas() error {
	//TODO: Just use the resource name
	resources := map[string]string{"agents": "agent", "offices": "office", "openhouses": "openhouse", "properties": "property"}
	schemaPref := root + "resources/es_mappings/es_"

	standardSchemas = map[string]map[string]empty{}
	customSchemas = map[string]map[string]schemaNest{}

	fin := make(chan error, 2)
	//process standard schema
	go func() {
		for resourceType := range resources {
			propertiesMap, err := readSchemaStandard(schemaPref + resourceType + "_standard.json")
			curSchem := map[string]empty{}
			if err != nil {
				fin <- err
				return
			}
			for k := range propertiesMap {
				curSchem[k] = empty{}
			}
			standardSchemas[resources[resourceType]] = curSchem
		}
		fin <- nil
	}()

	go func() {
		for resourceType := range resources {
			propertiesMap, err := readSchemaCustom(schemaPref + resourceType + "_custom.json")
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
			customSchemas[resources[resourceType]] = curSchem
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
			res = append(res, file.Name())
		}
	}
	return res, nil
}

//given a mapping json filename, returns resource and class
//TODO: make it less hacky?
func filenameChunker(filename string) (string, string) {
	res := strings.Split(filename, "_")
	return strings.TrimPrefix(res[1], "active"), strings.Join(res[2:], "_")
}

func checkRoutine(jsonMap string, fin chan bool, log *strings.Builder) {
	fmt.Fprintln(log, jsonMap)
	mapping, err := readJSON(root + "mappings/" + board + "/" + jsonMap)
	if err != nil {
		fmt.Fprintln(log, err)
		fin <- false
		return
	}

	//stores all non-nested values we have encountered so far
	mappedFieldvals := map[string]string{}
	resource, _ := filenameChunker(jsonMap)
	//TODO: Read csv metadata
	if mapping != nil {
		for key := range mapping {
			switch mapping[key].(type) {
			case string:
				mappedVal := mapping[key].(string)
				if mappedVal == "" {
					//ignore empty fields
					continue
				}
				//Check if another field is already mapped to the same thing
				other, ex := mappedFieldvals[mappedVal]
				if ex {
					fmt.Fprintln(log, "	", key+":", "repeated value", mappedVal, "with", other)
				} else {
					mappedFieldvals[mappedVal] = key
				}
				//Check if in custom schema. We do custom first because custom schema is smaller
				aNest, ex := customSchemas[resource][mappedVal]
				if !ex {
					//check if in standard schema
					_, ex := standardSchemas[resource][mappedVal]
					if !ex {
						fmt.Fprintln(log, "	", key+":", mappedVal, "is not in", resource+"'s", "standard nor custom schema")
					}
				} else if aNest.nested {
					fmt.Fprintln(log, "	", key+":", "is supposed to be a nest but was mapped to", mappedVal)
				}
			case interface{}:
				//Nested
				assertedNest := mapping[key].([]interface{})
				nestSchem, ok := assertedNest[0].(string)
				if !ok {
					fmt.Fprintln(log, "	", key+":", "Nesting", key, "is missing the custom type")
					continue
				}
				nesting, ok := assertedNest[1].(map[string]interface{})
				if !ok {
					fmt.Fprintln(log, "	", key+":", "Nesting", key, "is missing mappings")
					continue
				}
				nestKeyinside := false
				nestName := false
				nestType := false
				aNest, ex := customSchemas[resource][nestSchem]
				if !ex || !aNest.nested {
					fmt.Fprintln(log, "	", key+":", "Nesting", key, "has an invalid nesting", nestSchem)
				}
				for mapField := range nesting {
					switch mapField {
					case "Name":
						nestName = true
					case "Type":
						nestType = true
						typeString, ok := nesting[mapField].(string)
						if !ok {
							fmt.Fprintln(log, "	", key+":", "Nesting", key, "has empty Type")
						}
						curAdt, ex := adt[nestSchem]
						if ex {
							typeList := strings.Split(typeString, ",")
							for _, aType := range typeList {
								if curAdt[sort.SearchStrings(curAdt, strings.Trim(aType, " "))] != strings.Trim(aType, " ") {
									fmt.Fprintln(log, "	", key+":", "Nesting", key, "has type", aType, "which is illegal for", nestSchem)
								}
							}
						}
					default:
						if mapField == key {
							nestKeyinside = true
						}
						_, ex = customSchemas[resource][nestSchem].properties[nesting[mapField].(string)]
						if !ex {
							fmt.Fprintln(log, "	", key+":", "Nested property", mapField, "has an invalid nesting", nesting[mapField].(string), "for", nestSchem)
						}
						//TODO: use ADT here
						//fmt.Println(mapping[key])
					}
				}
				if !nestName {
					fmt.Fprintln(log, "	", key+":", "Missing Name inside nesting")
				}
				if !nestType {
					fmt.Fprintln(log, "	", key+":", "Missing Type inside nesting")
				}
				if !nestKeyinside {
					fmt.Fprintln(log, "	", key+":", "Missing itself inside nesting")
				}
			default:
				fmt.Fprintln(log, "	", key+":", "Unknown mapping")
			}
		}
	} else {
		fmt.Fprintln(log, "Mapping is nil!")
	}
	//TODO: Check keys missing or key not from metadata
	fin <- true
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
	//TODO: Load acceptable data types
	err = readSchemas()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if adt, err = readAdt(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	mappingsList, err := getFilesInDir(root + "mappings/" + board + "/")
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	finChan := make(chan bool, len(mappingsList))
	loggers := make([]*strings.Builder, len(mappingsList))
	//spin off analysis of each mapping in their own goroutine
	for i, jsonMap := range mappingsList {
		loggers[i] = new(strings.Builder)
		go checkRoutine(jsonMap, finChan, loggers[i])
	}

	//wait for all to finish
	for range mappingsList {
		select {
		case <-finChan:
		}
	}

	//Print out all logs in order
	for _, log := range loggers {
		fmt.Print(log.String())
	}
}
