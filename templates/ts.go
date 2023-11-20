package templates

import (
	"os"
	"strings"
	"text/template"

	"github.com/andrewwormald/gobridge/reader"
)

type TSService struct {
	Name        string
	Interfaces  []TSInterface
	Enums       []TSEnum
	ModName     string
	MethodNames []string
}

type TSInterface struct {
	Name   string
	Fields []reader.TypeSignature
}

type TSEnum struct {
	Name   string
	Fields map[string]string
}

func (tss *TSService) AddTo(file *os.File) error {
	funcMap := template.FuncMap{
		"ToLower": strings.ToLower,
		"ToCamelCase": func(s string) string {
			ls := strings.Split(s, "")
			ls[0] = strings.ToUpper(ls[0])
			return strings.Join(ls, "")
		},
	}

	return template.Must(template.New("").Funcs(funcMap).Parse(tsServiceTemplate)).Execute(file, tss)
}

var tsServiceTemplate = `import { Injectable } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { environment } from '../../environments/environment';

@Injectable({
  providedIn: 'root'
})
export class {{.Name}} {

  constructor(private http: HttpClient) {}
  {{- range $key, $value := .MethodNames }}

  // @ts-ignore
  public async {{$value}}(payload: {{$value}}Request): Promise<{{$value}}Response> {
    // tslint:disable-next-line:max-line-length
    return await this.http.post(environment.BackendURL + '/{{$.Name | ToLower}}/{{$value | ToLower}}', JSON.stringify(payload)).toPromise() as {{$value}}Response;
  }

{{- end }}
}
{{- range $key, $value := .Interfaces }}
{{- if eq (len $value.Fields) 0 }}

// tslint:disable-next-line:no-empty-interface
export interface {{$value.Name}} {}
{{- end}}
{{- if ge (len $value.Fields) 1 }}

export interface {{$value.Name}} {
{{- range $key2, $value2 := $value.Fields }}
{{- if eq $value2.Type 1}}
  {{ $value2.Name | ToCamelCase}}: {{ $value2.Kind }};
{{- end}}
{{- if eq $value2.Type 0}}
  {{ $value2.Name | ToCamelCase}}: {{ $value2.Kind }};
{{- end}}
{{- if eq $value2.Type 2}}
  {{ $value2.Name | ToCamelCase}}: {{ $value2.Kind }}[];
{{- end}}
{{- end }}
}
{{- end}}
{{- end }}

{{- range $key, $value := .Enums }}
{{- if eq (len $value.Fields) 0 }}

export enum {{$value.Name}} {}
{{- end}}
{{- if ge (len $value.Fields) 1 }}

export enum {{$value.Name}} {
{{- range $key2, $value2 := $value.Fields }}
  {{ $key2 }} = {{ $value2 }},
{{- end }}
}
{{- end}}
{{- end }}
`
