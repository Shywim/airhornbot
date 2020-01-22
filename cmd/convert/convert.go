package main

import (
	"flag"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/jonas747/dca"
)

var filePath string

func main() {
	flag.StringVar(&filePath, "filepath", "", "Path to the file to convert")
	flag.Parse()
	println(filePath)
	if filePath == "" {
		panic("The path cannot be empty!")
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		panic("The file doesn't exist")
	}

	dcaSession, err := dca.EncodeFile(filePath, dca.StdEncodeOptions)
	defer dcaSession.Cleanup()
	if err != nil {
		panic(err)
	}

	out, err := os.Create(strings.TrimSuffix(filepath.Base(filePath), path.Ext(filePath)) + ".dca")
	if err != nil {
		panic(err)
	}

	io.Copy(out, dcaSession)
	println("Successfully converted file to DCA")
}
