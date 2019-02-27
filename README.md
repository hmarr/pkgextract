# pkgextract

A library for extracting a set of Go packages that depend on one another.

## Example

Extract the `cmd/go/internal/modload` package from [golang/go](https://github.com/golang/go), along with all the package it depends on, replacing `internal` with `_internal_` in import paths.

```go
package main

import (
	"log"
	"path/filepath"
	"strings"

	"github.com/hmarr/pkgextract"
)

func main() {
	extractor := pkgextract.NewPackageExtractor(pkgextract.PackageExtractorOptions{
		InitialPkg: "cmd/go/internal/modload",
		OutputRoot: "./out",
		ImportRewriter: func(importPath string) string {
			newPath := strings.Replace(importPath, "internal", "_internal_", -1)
			return filepath.Join("github.com/dependabot/gomodules-extracted", newPath)
		},
		Filter: func(importPath string) bool {
			return strings.Contains(importPath, "internal")
		},
	})

	if err := extractor.Run(); err != nil {
		log.Fatal(err)
	}
}
```
