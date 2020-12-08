package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"

	edid "github.com/thyge/goedid"
)

var (
	edidfile = flag.String("edidfile", "./edid.bin", "path to edid file")
)

func main() {
	flag.Parse()

	fileExtension := filepath.Ext(*edidfile)
	edidBytes, err := ioutil.ReadFile(*edidfile)
	if err != nil {
		log.Fatal("Unable to read file ", err)
	}
	if fileExtension == ".txt" {
		edidBytes, err = GetBytesFromString(string(edidBytes))
	}
	if err != nil {
		log.Fatal("Unable to read file ", err)
	}

	decodedEDID, err := edid.DecodeEDID(edidBytes)
	if err != nil {
		log.Fatal("Unable to decode EDID ", err)
	}
	// pretty print json version of edid structure
	pretty, err := json.MarshalIndent(decodedEDID, "", "    ")
	fmt.Println(string(pretty))
}

func GetBytesFromString(str string) ([]byte, error) {
	str = strings.Replace(str, " ", "", -1)
	str = strings.Replace(str, "\r\n", "", -1)
	str = strings.Replace(str, "\n", "", -1)
	str = strings.TrimSpace(str)
	return hex.DecodeString(str)
}
