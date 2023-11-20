package generator

import (
	"fmt"
	"math/rand"
	"os"
	"strings"

	"github.com/andrewwormald/gobridge/ioeasy"
	"github.com/andrewwormald/gobridge/reader"
	"github.com/andrewwormald/gobridge/templates"
)

func TSClient(tsPath, serviceName string, d *reader.Data) error {
	err := ioeasy.CreateFileFromPath(tsPath)
	if err != nil {
		return err
	}

	// Reset file
	err = os.Truncate(tsPath, 0)
	if err != nil {
		return err
	}

	// Apply correct permissions
	file, err := os.OpenFile(tsPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	rawTypes := d.GoTypeRep
	fs := d.APIFuncs

	var tsdata []reader.GoTypeRepresentation
	for _, v := range rawTypes {
		if v.Type == reader.GenericTypeStruct {
			v.Fields = parseTSTypes(v.Fields)
			tsdata = append(tsdata, v)
		}

		if v.Type == reader.GenericTypeEnum {
			v.Kind = switchToTypescriptType(v.Kind)
			tsdata = append(tsdata, v)
		}
	}

	tsi := new(templates.TSService)
	tsi.Name = serviceName
	for _, methods := range fs {
		for _, m := range methods {
			tsi.MethodNames = append(tsi.MethodNames, m.Name)

			req := templates.TSInterface{
				Name:   m.Name + "Request",
				Fields: parseTSTypes(m.Params),
			}
			tsi.Interfaces = append(tsi.Interfaces, req)

			resp := templates.TSInterface{
				Name:   m.Name + "Response",
				Fields: parseTSTypes(m.Results),
			}

			tsi.Interfaces = append(tsi.Interfaces, resp)
		}
	}

	for _, v := range tsdata {
		switch v.Type {
		case reader.GenericTypeStruct:
			tst := templates.TSInterface{
				Name:   v.Name,
				Fields: v.Fields,
			}

			tsi.Interfaces = append(tsi.Interfaces, tst)

		case reader.GenericTypeEnum:
			tst := templates.TSEnum{
				Name:   v.Name,
				Fields: make(map[string]string),
			}

			l := d.ValueDecl[v.Name]
			for _, decl := range l {
				for key, value := range decl {
					tst.Fields[key] = value
				}
			}

			tsi.Enums = append(tsi.Enums, tst)
		}
	}

	err = tsi.AddTo(file)
	if err != nil {
		return err
	}

	return nil
}

func parseTSTypes(m []reader.TypeSignature) []reader.TypeSignature {
	temp := make([]reader.TypeSignature, len(m))
	for i, v := range m {
		v.Kind = switchToTypescriptType(v.Kind)
		temp[i] = v
	}

	return temp
}

func GoClient(data map[string]map[string]string) (fContents string, err error) {
	return
}

func Server(serverPath, modName string, d *reader.Data) error {
	err := ioeasy.CreateFileFromPath(serverPath)
	if err != nil {
		return err
	}

	// Reset file
	err = os.Truncate(serverPath, 0)
	if err != nil {
		return err
	}

	// Apply correct permissions
	file, err := os.OpenFile(serverPath, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		return err
	}

	structs := d.GoTypeRep
	apiPkgName := d.ApiPkgName

	mrs := make(map[string]reader.GoTypeRepresentation)
	for _, v := range structs {
		mrs[v.Name] = v
	}

	apiImport := d.ImportDictionary[apiPkgName]
	var additionalImports []string

	fs := d.APIFuncs
	pkgToImport := d.ImportDictionary

	for api, funcs := range fs {
		var (
			hs []templates.HTTPHandler
			ps []templates.Path
		)
		for _, fn := range funcs {
			p := templates.Path{
				Camelcase: fn.Name,
				Lowercase: apiPkgName + "/" + strings.ToLower(fn.Name),
			}

			var (
				params         []string
				results        []string
				responseParams []string
				ts             templates.SerialisationTypes
			)

			ts.Request = fn.Params
			for i, val := range fn.Params {
				if val.Name == "" && isBuiltInType(val.Kind) {
					params = append(params, RandVarName())
				} else {
					cleaned := strings.ReplaceAll(val.Kind, "[]", "")
					t, ok := mrs[cleaned]
					if ok {
						fn.Params[i].GoPackage = t.Pkg
						if imp, exists := pkgToImport[t.Pkg]; exists {
							if imp != apiImport {
								additionalImports = append(additionalImports, imp)
							}
						}
					} else {
						// Core library support
						if !isBuiltInType(val.Kind) {
							pkg := strings.Split(val.Kind, ".")[0]
							fn.Params[i].GoPackage = pkg
							fn.Params[i].Kind = strings.Split(val.Kind, ".")[1]
							if _, exists := pkgToImport[pkg]; !exists {
								if val.Name != apiImport {
									additionalImports = append(additionalImports, pkg)
									pkgToImport[pkg] = pkg
								}
							} else if imp, exists := pkgToImport[pkg]; !exists {
								if imp != apiImport {
									additionalImports = append(additionalImports, imp)
									pkgToImport[pkg] = pkg
								}
							}
						}
					}
					params = append(params, val.Name)
				}
			}

			ts.Response = fn.Results
			for i, val := range fn.Results {
				if isBuiltInType(val.Kind) {
					results = append(results, RandVarName())
				} else {
					cleaned := strings.ReplaceAll(val.Kind, "[]", "")
					t, ok := mrs[cleaned]
					if ok {
						fn.Results[i].GoPackage = t.Pkg
						if imp, exists := pkgToImport[t.Pkg]; exists {
							if imp != apiImport {
								additionalImports = append(additionalImports, imp)
							}
						}
					}
					results = append(results, val.Name)
				}
				responseParams = append(responseParams, val.Name)
			}
			responseParams = append(responseParams, "_")

			results = append(results, "err")

			h := templates.HTTPHandler{
				Method:         fn.Name,
				API:            apiPkgName + "." + api,
				URL:            apiPkgName + "/" + strings.ToLower(fn.Name),
				RequestType:    fn.Name,
				Params:         params,
				Results:        results,
				ResponseType:   fn.Name,
				ResponseParams: responseParams,
				Types:          ts,
			}

			ps = append(ps, p)
			hs = append(hs, h)
		}

		server := &templates.HTTPServer{
			API:      apiPkgName + "." + api,
			Imports:  append(additionalImports, apiImport),
			Paths:    ps,
			Handlers: hs,
		}

		return server.AddTo(file)
	}

	return nil
}

func isBuiltInType(typ string) bool {
	if strings.HasPrefix(typ, "[]") {
		return true
	}

	if strings.HasPrefix(typ, "map") {
		return true
	}

	switch typ {
	case "bool", "byte", "complex128", "complex64", "error":
	case "float32", "float64":
	case "int", "int16", "int32", "int64", "int8":
	case "rune", "string":
	case "uint", "uint16", "uint32", "uint64", "uint8", "uintptr":
	default:
		return false
	}
	return true
}

func switchToTypescriptType(typ string) string {
	switch typ {
	case "byte", "complex128", "complex64", "error":
		return "string"
	case "bool":
		return "boolean"
	case "float32", "float64":
		return "number"
	case "int", "int16", "int32", "int64", "int8":
		return "number"
	case "rune", "string":
		return "string"
	case "uint", "uint16", "uint32", "uint64", "uint8", "uintptr":
		return "number"
	case "Time", "time.Time":
		return "Date"
	default:
		// Consider and treat it as a non-primitive type

		// Trade Go slices for TS arrays
		if strings.HasPrefix(typ, "[]") {
			typ = strings.TrimPrefix(typ, "[]")
			typ = switchToTypescriptType(typ)
			typ += "[]"
		}

		// Trade Go maps for TS any as there is no easy swap
		if strings.HasPrefix(typ, "map") { // map[int64]bool
			typ = strings.TrimPrefix(typ, "map[") // int64]bool

			keyAndValueTypes := []string{}              // 0 - key, 1 - value of a map
			for _, v := range strings.Split(typ, "]") { // [int64, bool]
				keyAndValueTypes = append(keyAndValueTypes, switchToTypescriptType(v))
			}

			typ = fmt.Sprintf("Record<%s, %s>", keyAndValueTypes[0], keyAndValueTypes[1])
		}

		return typ
	}
}

func RandVarName() string {
	var availableChars = []rune("abcdefghijklmnopqrstuvwxyz")

	b := make([]rune, 4)
	for i := range b {
		b[i] = availableChars[rand.Intn(len(availableChars))]
	}
	return string(b)
}
