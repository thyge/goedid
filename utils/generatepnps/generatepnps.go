package main

// Idea from: https://github.com/jojonas/pyedid/blob/master/pyedid/helpers/registry.py

import (
	"encoding/csv"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	pnpfile = flag.String("ceafile", "./PNP.csv", "path to csv file")
)

func main() {
	in, err := ioutil.ReadFile(*pnpfile)
	f, err := os.Create("../../edid/pnps.go")
	if err != nil {
		return
	}
	r := csv.NewReader(strings.NewReader(string(in)))

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}
	f.WriteString("package edid\n")
	f.WriteString("\n")
	f.WriteString("var pnpLookup = map[string]PNPID{\n")
	for i := 1; i < len(records); i++ {
		f.WriteString("\t" + `"` + records[i][1] + `"` + ": PNPID{\n")
		f.WriteString("\t\t" + "ID: " + `"` + records[i][1] + `"` + ",\n")
		f.WriteString("\t\t" + "Company: " + `"` + records[i][0] + `"` + ",\n")
		f.WriteString("\t\t" + "Date: " + `"` + records[i][2] + `"` + ",\n")
		f.WriteString("\t" + "},\n")
	}
	f.WriteString("}\n")
}
