package main

import (
	"archive/zip"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lftk/anki"
)

func main() {
	var dir, output string
	switch args := os.Args[1:]; len(args) {
	case 1:
		dir = args[0]
		output = filepath.Base(dir)
	case 2:
		dir = args[0]
		output = args[1]
	default:
		fmt.Println("usage: colpkg <dir> [<name>]")
		os.Exit(1)
	}

	if err := pack(dir, output); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func pack(dir, output string) error {
	if !strings.HasSuffix(output, ".colpkg") {
		output += ".colpkg"
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	path := filepath.Join(wd, output)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	z := zip.NewWriter(f)
	defer z.Close()

	return anki.Pack(z, dir)
}
