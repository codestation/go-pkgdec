package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"

	"megpoid.xyz/go/go-pkgdec/pkg"
	"net/url"
)

func checkFatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func isValidUrl(toTest string) bool {
	u, err := url.ParseRequestURI(toTest)

	if err == nil && u.Scheme != "" {
		return true
	} else {
		return false
	}
}

func main() {
	input := flag.String("i", "", "Package file or URL (required)")
	license := flag.String("l", "", "License in zRIF format")
	output := flag.String("o", "", "Directory to extract the files")
	zipped := flag.Bool("z", false, "Create a zipfile from the pkg file")

	flag.Parse()

	if *input == "" && *output == "" {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	var r *pkg.Reader
	var err error

	if isValidUrl(*input) {
		response, err := http.Get(*input)
		checkFatal(err)
		defer response.Body.Close()

		r, err = pkg.NewReader(response.Body, *license)
		checkFatal(err)
	} else {
		f, err := os.Open(*input)
		checkFatal(err)
		defer f.Close()

		r, err = pkg.NewReader(f, *license)
		checkFatal(err)
	}

	title := r.GetTitle()
	fmt.Printf("Unpacking %s\n", title)

	if !*zipped {
		err = r.Unpack(*output)
	} else {
		err = r.CreateZip(*output)
	}

	checkFatal(err)

	if r.Valid() {
		fmt.Printf("PKG hash check OK\n")
	} else {
		fmt.Printf("PKG SHA1 check failed\n")
		fmt.Printf("Actual:   %x\n", r.CalculatedHash)
		fmt.Printf("Expected: %x\n", r.FileHash)
	}
}
