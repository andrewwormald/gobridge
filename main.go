package main

import (
	"flag"
	"math/rand"

	"github.com/luno/gobridge/generator"
	"github.com/luno/gobridge/reader"
)

var inputFile = flag.String("api", "", "Target file to read")
var moduleName = flag.String("mod", "", "Name of the go module being used for the project")
var tsOutFile = flag.String("ts", "", "Target location to generate file to read")
var tsServiceName = flag.String("ts_service", "", "Target location to generate file to read")
var goServerFile = flag.String("server", "", "")

func main() {
	flag.Parse()
	rand.Seed(123456789)

	if *inputFile == "" || *moduleName == "" {
		return
	}

	d, err := reader.ParseFile(*inputFile, *moduleName)
	if err != nil {
		panic(err)
	}

	if *tsOutFile != "" {
		err := generator.TSClient(*tsOutFile, *tsServiceName, d)
		if err != nil {
			panic(err)
		}
	}

	if *goServerFile != "" {
		err = generator.Server(*goServerFile, *moduleName, d)
		if err != nil {
			panic(err)
		}
	}
}
