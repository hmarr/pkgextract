package pkgextract

import (
	"bufio"
	"container/list"
	"go/ast"
	"go/build"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
)

type rewriterFunc func(importPath string) string
type filterFunc func(importPath string) bool

// NewPackageExtractor builds a new package extractor, given a set of options
func NewPackageExtractor(opts PackageExtractorOptions) *PackageExtractor {
	return &PackageExtractor{
		PackageExtractorOptions: opts,
		importMap:               make(map[string]string),
	}
}

// PackageExtractor extracts a set of dependent packages
type PackageExtractor struct {
	PackageExtractorOptions
	importMap map[string]string
}

// PackageExtractorOptions let you configure the package extractor
type PackageExtractorOptions struct {
	InitialPkg     string
	ImportRewriter rewriterFunc
	OutputRoot     string
	Filter         filterFunc
}

// Run executes the package extraction process
func (pe *PackageExtractor) Run() error {
	pe.importMap[pe.InitialPkg] = pe.ImportRewriter(pe.InitialPkg)
	pe.scanForPackages()

	for originalPath := range pe.importMap {
		if err := pe.extractPackage(originalPath); err != nil {
			return err
		}
	}

	return nil
}

// scanForPackages finds packages to be extracted, and generates new import
// paths for them
func (pe *PackageExtractor) scanForPackages() error {
	buildCtx := build.Default

	toScan := list.New()
	toScan.PushBack(pe.InitialPkg)
	seen := map[string]bool{pe.InitialPkg: true}
	for toScan.Len() > 0 {
		pkgPath := toScan.Front()
		toScan.Remove(pkgPath)

		pkg, err := buildCtx.Import(pkgPath.Value.(string), "", 0)
		if err != nil {
			return err
		}

		for _, impPath := range pkg.Imports {
			if _, ok := seen[impPath]; ok {
				continue
			}

			if pe.Filter(impPath) {
				pe.importMap[impPath] = pe.ImportRewriter(impPath)
				toScan.PushBack(impPath)
			}
			seen[impPath] = true
		}
	}
	return nil
}

// extractPackage extracts a given package to the output directory
func (pe *PackageExtractor) extractPackage(importPath string) error {
	ctx := build.Default
	pkg, err := ctx.Import(importPath, "", 0)
	if err != nil {
		return err
	}

	newImportPath := pe.ImportRewriter(importPath)
	os.MkdirAll(filepath.Join(pe.OutputRoot, newImportPath), 0755)

	srcDir := filepath.Join(pkg.SrcRoot, importPath)

	for _, f := range pkg.GoFiles {
		srcPath := filepath.Join(srcDir, f)
		dstPath := filepath.Join(pe.OutputRoot, newImportPath, f)
		err := pe.extractFile(srcPath, dstPath)
		if err != nil {
			return err
		}
	}
	return nil
}

// extractFile extracts a file to the output directory, rewriting any imports
// that reference other extracted packages
func (pe *PackageExtractor) extractFile(srcPath, dstPath string) error {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, srcPath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	if err := pe.updateImports(f); err != nil {
		return err
	}

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	w := bufio.NewWriter(out)
	if err := printer.Fprint(w, fset, f); err != nil {
		return err
	}
	if err := w.Flush(); err != nil {
		return err
	}

	return nil
}

func (pe *PackageExtractor) updateImports(f *ast.File) error {
	for _, impSpec := range f.Imports {
		impPath, err := strconv.Unquote(impSpec.Path.Value)
		if err != nil {
			return err
		}

		if newPath, ok := pe.importMap[impPath]; ok {
			impSpec.Path.Value = strconv.Quote(newPath)
		}
	}
	return nil
}
