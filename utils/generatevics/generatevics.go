package main

import (
	"encoding/csv"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

var (
	ceafile = flag.String("ceafile", "./vics.csv", "path to csv file")
)

func main() {
	in, err := ioutil.ReadFile(*ceafile)
	f, err := os.Create("vics.go")
	if err != nil {
		return
	}
	r := csv.NewReader(strings.NewReader(string(in)))

	records, err := r.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	// file header
	f.WriteString("package edid\n")
	f.WriteString("\n")
	f.WriteString("var vicLooup = map[byte]CEAResulution{\n")
	for i := 1; i < len(records); i++ {
		f.WriteString("\t" + records[i][0] + ": CEAResulution{\n")
		f.WriteString("\t\tVIC:" + records[i][0] + ",\n")
		f.WriteString("\t\tName:" + `"` + records[i][1] + `"` + ",\n")
		f.WriteString("\t\tDescription:" + `"` + records[i][2] + `"` + ",\n")
		f.WriteString("\t\tPixelMHz:" + records[i][6] + ",\n")
		f.WriteString("\t\tHorizontalActive:" + records[i][3] + ",\n")
		f.WriteString("\t\tVerticalActive:" + records[i][4] + ",\n")
		f.WriteString("\t\tNative:" + `"` + records[i][9] + `"` + ",\n")
		f.WriteString("\t},\n")
	}
	f.WriteString("}\n")

	f.Sync()
}
