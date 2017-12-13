package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"megpoid.xyz/go/go-pkgdec/pkg"
)

func checkFatal(err error) {
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func main() {
	license := flag.String("l", "", "License in zRIF format")
	input := flag.String("i", "", "Package file")
	output := flag.String("o", "", "Directory created to extract the files")

	flag.Parse()

	if *input == "" && *license == "" {
		fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(1)
	}

	if *input != "" && *license != "" {
		checkFatal(errors.New("use either a zRIF license or a license file"))
	}

	if *license != "" {
		lic, err := pkg.DecodeLicense(*license, 0)
		checkFatal(err)

		if *output != "" {
			err = ioutil.WriteFile(*output, lic, 0644)
			checkFatal(err)
		} else {
			os.Stdout.Write(lic)
		}
	} else if *input != "" {
		lic, err := ioutil.ReadFile(*input)
		checkFatal(err)

		rif, err := pkg.EncodeLicense(lic)
		checkFatal(err)

		fmt.Println(rif)
	}
}
