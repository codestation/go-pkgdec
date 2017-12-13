go-pkgdec
========

Decrypt and extract PKG files

The `pkgdec` command runs the same across platform and has no external dependencies

## Install

```bash
go get megpoid.xyz/go/go-pkgdec/cmd/pkgdec
```

## Command Use

Decrypt and unpack a PKG file:

```bash
$ pkgdec -i <file.pkg or http://host/file.pkg> [-o <output dir>] [-l <zRIF string>]
```

(The license is required to generate the work.bin file)

## Library Use

```go
package main

import "log"
import "os"
import "megpoid.xyz/go/go-pkgdec/pkg"

func main() {
    r, err := pkg.OpenReader("file.pkg", "")
    if err != nil {
        log.Fatal(err)
    }
    defer r.Close()
    
    err = r.Unpack("output_folder", nil)
    if err != nil {
        log.Fatal(err)
    }
    
    if ! r.Valid() {
        log.Println("PKG hash check failed")
    }
}
```

## Why another unpacker

No reason. Just a quick project to improve my Go learning

## License

MIT
