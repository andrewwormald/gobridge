package reader

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/ioutil"
	"strings"
)

type Data struct {
	GoTypeRep        []GoTypeRepresentation
	APIFuncs         map[string][]FunctionSignature
	ApiPkgName       string
	ImportDictionary map[string]string              // Package name to import path from go mod
	ValueDecl        map[string][]map[string]string // Constants and var declarations found
}

func ParseFile(path string, mName string) (*Data, error) {
	p1 := strings.Split(path, ".go")
	p2 := strings.Split(p1[0], "/")

	dir := strings.TrimSuffix(path, "/"+strings.Join(p2[len(p2)-1:], "")+".go")
	dirInfo, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var filesToRead []string
	for _, v := range dirInfo {
		if v.IsDir() {
			continue
		}

		filesToRead = append(filesToRead, dir+"/"+v.Name())
	}

	fset := token.NewFileSet()
	d := &Data{
		APIFuncs:         make(map[string][]FunctionSignature),
		ImportDictionary: make(map[string]string),
		ValueDecl:        make(map[string][]map[string]string),
	}

	for _, filePath := range filesToRead {
		err = readFile(fset, mName, filePath, d, true)
		if err != nil {
			return nil, err
		}
	}

	return d, nil
}

func readFile(fset *token.FileSet, mName string, filePath string, d *Data, recursive bool) error {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	node, err := parser.ParseFile(fset, "", string(b), parser.ParseComments)
	if err != nil {
		return err
	}

	r := &Reader{
		fset: fset,
	}

	var sp []TypeSignature
	// Traverse file tree for interface methods
	ast.Inspect(node, func(n ast.Node) bool {
		switch t := n.(type) {
		case *ast.File:
			r.CurrGoPkg = t.Name.Name
			d.ImportDictionary[r.CurrGoPkg] = mName + getDirFromPath(filePath)
		case *ast.TypeSpec:
			if t.Name.IsExported() {
				switch assertion := t.Type.(type) {
				case *ast.StructType:
					sp = r.ListStructProperties(assertion)
					d.GoTypeRep = append(d.GoTypeRep, GoTypeRepresentation{
						Name:   t.Name.Name,
						Pkg:    r.CurrGoPkg,
						Type:   GenericTypeStruct,
						Fields: sp,
					})
				case *ast.InterfaceType:
					if recursive == true {
						apiFs := r.ListInterfaceMethods(assertion)
						d.APIFuncs[t.Name.Name] = apiFs
						d.ApiPkgName = r.CurrGoPkg
					}
				case *ast.Ident:
					// Possible enum of a primitive type
					d.GoTypeRep = append(d.GoTypeRep, GoTypeRepresentation{
						Name: t.Name.Name,
						Pkg:  r.CurrGoPkg,
						Type: GenericTypeEnum,
						Kind: assertion.Name,
					})
				}
			}
		case *ast.ValueSpec:
			_, ok := t.Values[0].(*ast.BasicLit)
			if ok {
				m := make(map[string]string)
				m[t.Names[0].Name] = t.Values[0].(*ast.BasicLit).Value

				_, isIdent := t.Type.(*ast.Ident)
				if isIdent {
					d.ValueDecl[t.Type.(*ast.Ident).Name] = append(d.ValueDecl[t.Type.(*ast.Ident).Name], m)
				}
			}
		case *ast.ImportSpec:
			if !strings.Contains(t.Path.Value, mName) {
				return false
			}

			if !recursive {
				return false
			}

			dirLvlCorrection := ""
			path := strings.ReplaceAll(t.Path.Value, `"`, "")
			if strings.HasPrefix(path, mName) {
				path = strings.ReplaceAll(path, mName, "")
			} else {
				return false
			}

			pathList := strings.Split(path, "/")
			path = strings.Join(pathList[1:], "/")
			dirLvlCorrection = strings.Split(filePath, strings.Split(path, "/")[0])[0]

			fi, err := ioutil.ReadDir(dirLvlCorrection + path)
			if err != nil {
				panic(err)
			}

			for _, f := range fi {
				if f.IsDir() {
					continue
				}

				nextPath := dirLvlCorrection + path + "/" + f.Name()
				err = readFile(fset, mName, nextPath, d, false)
				if err != nil {
					panic(err)
				}
			}
		}
		return true
	})

	return nil
}

func getDirFromPath(path string) string {
	var dir string
	for _, item := range strings.Split(path, "/") {
		if item == "." {
			continue
		}

		if item == ".." {
			continue
		}

		if strings.HasSuffix(item, ".go") {
			continue
		}

		dir += "/" + item
	}
	return dir
}

type Reader struct {
	fset      *token.FileSet
	CurrGoPkg string
}

// ListInterfaceMethods returns the function signatures of all the interface methods
func (r *Reader) ListStructProperties(it *ast.StructType) []TypeSignature {
	var sf []TypeSignature
	for _, field := range it.Fields.List {
		// Name
		ts := TypeSignature{}
		ts.Name = field.Names[0].Name
		// Kind
		switch t := field.Type.(type) {
		case *ast.Ident:
			ts.Kind = t.Name
		case *ast.SelectorExpr:
			p2 := t.Sel
			ts.Kind = p2.Name
		case *ast.ArrayType:
			ts.Kind = "[]" + r.importTypeFromASTExpr(t.Elt)
		case *ast.MapType:
			p1 := r.importTypeFromASTExpr(t.Key)
			p2 := r.importTypeFromASTExpr(t.Value)
			ts.Kind = fmt.Sprintf("map[%s]%s", p1, p2)
		default:
			fmt.Println("Unable to determine type:")
			ast.Print(r.fset, t)
		}

		sf = append(sf, ts)
	}
	return sf
}

type GenericType int

const (
	GenericTypeUnknown GenericType = 0
	GenericTypeStruct  GenericType = 1
	GenericTypeEnum    GenericType = 2
)

type GoTypeRepresentation struct {
	Name       string
	Pkg        string
	ImportPath string
	Fields     []TypeSignature // This will only have a value if the GenericType is set to Struct
	Type       GenericType
	Kind       string // This will only have a value if the GenericType is set to Enum
}

type TypeSignature struct {
	Name      string
	Kind      string
	Type      SignatureType
	GoPackage string
}

type SignatureType int

const (
	SignatureTypeUnknown SignatureType = 0
	SignatureTypeSingle  SignatureType = 1
	SignatureTypeSlice   SignatureType = 2
)

type FunctionSignature struct {
	Name    string
	Params  []TypeSignature
	Results []TypeSignature
}

// ListInterfaceMethods returns the function signatures of all the interface methods
func (r *Reader) ListInterfaceMethods(it *ast.InterfaceType) []FunctionSignature {
	var fsSlice []FunctionSignature

	methods := it.Methods.List
	for _, method := range methods {
		fn, ok := method.Type.(*ast.FuncType)
		if !ok {
			continue
		}

		sig := r.CheckFunctionSignature(fn)
		sig.Name = method.Names[0].Name
		fsSlice = append(fsSlice, sig)
	}

	return fsSlice
}

func (r *Reader) CheckFunctionSignature(fn *ast.FuncType) FunctionSignature {
	var fs FunctionSignature

	for _, param := range fn.Params.List {
		n, t := r.parseVariables(param)
		if strings.Contains(strings.ToLower(t), "context") ||
			strings.Contains(strings.ToLower(t), "error") {
			continue
		}

		if strings.HasPrefix(n, "[]") {
			n = "list"
		}

		typ := SignatureTypeSingle
		if strings.HasPrefix(t, "[]") {
			typ = SignatureTypeSlice
			t = strings.ReplaceAll(t, "[]", "")
		}

		fs.Params = append(fs.Params, TypeSignature{
			Name: n,
			Kind: t,
			Type: typ,
		})
	}

	for _, result := range fn.Results.List {
		n, v := r.parseVariables(result)
		if strings.Contains(strings.ToLower(v), "context") ||
			strings.Contains(strings.ToLower(v), "error") {
			continue
		}

		if strings.HasPrefix(n, "[]") {
			n = "list"
		}

		typ := SignatureTypeSingle
		if strings.HasPrefix(v, "[]") {
			typ = SignatureTypeSlice
			v = strings.ReplaceAll(v, "[]", "")
		}

		fs.Results = append(fs.Results, TypeSignature{
			Name: n,
			Kind: v,
			Type: typ,
		})
	}

	return fs
}

func (r *Reader) parseVariables(field *ast.Field) (name string, importTyp string) {
	if len(field.Names) < 1 {
		return r.importTypeFromASTExpr(field.Type), r.importTypeFromASTExpr(field.Type)
	}

	return field.Names[0].Name, r.importTypeFromASTExpr(field.Type)
}

func (r *Reader) importTypeFromASTExpr(expr ast.Expr) string {
	switch s := expr.(type) {
	case *ast.Ident:
		return importTypeFromASTIdent(s)
	case *ast.SelectorExpr:
		return importTypeFromASTIdent(s.X.(*ast.Ident)) + "." + importTypeFromASTIdent(s.Sel)
	case *ast.ArrayType:
		return "[]" + r.importTypeFromASTExpr(s.Elt)
	case *ast.MapType:
		p1 := r.importTypeFromASTExpr(s.Key)
		p2 := r.importTypeFromASTExpr(s.Value)
		return fmt.Sprintf("map[%s]%s", p1, p2)
	default:
		return ""
	}
}

// importTypeFromASTIdent collects the type name and can be used for non-import types in the file tree
func importTypeFromASTIdent(ident *ast.Ident) string {
	return ident.Name
}
