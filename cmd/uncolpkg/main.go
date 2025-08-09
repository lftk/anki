package main

import (
	"archive/zip"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lftk/anki"
)

func main() {
	var path, output string
	switch args := os.Args[1:]; len(args) {
	case 1:
		path = args[0]
		output = strings.TrimSuffix(
			filepath.Base(path), filepath.Ext(path),
		)
	case 2:
		path = args[0]
		output = args[1]
	default:
		fmt.Println("usage: uncolpkg <path> [<dir>]")
		os.Exit(1)
	}

	if err := unpack(path, output); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func unpack(path, output string) error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}

	dir := filepath.Join(wd, output)
	if _, err = os.Stat(dir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err = os.Mkdir(dir, 0755); err != nil {
			return err
		}
	} else {
		// TODO: check if dir is empty
		return fmt.Errorf("%q already exists.", output)
	}

	z, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer z.Close()

	return anki.Unpack(&z.Reader, dir)
}
