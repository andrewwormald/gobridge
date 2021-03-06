package templates

import (
	"gobridge/reader"
	"os"
	"strings"
	"text/template"
)

type HTTPServer struct {
	API      string
	Imports  []string
	Paths    []Path
	Handlers []HTTPHandler
}

type Path struct {
	Camelcase string
	Lowercase string
}

type SerialisationTypes struct {
	Response []reader.TypeSignature
	Request  []reader.TypeSignature
}

type HTTPHandler struct {
	Method         string
	API            string
	URL            string
	RequestType    string
	Params         []string
	Results        []string
	ResponseType   string
	ResponseParams []string
	Types          SerialisationTypes
}

func (s *HTTPServer) AddTo(file *os.File) error {
	funcMap := template.FuncMap{
		"ToCamelCase": func(s string) string{
			ls := strings.Split(s, "")
			ls[0] = strings.ToUpper(ls[0])
			return strings.Join(ls, "")
		},
	}
	return template.Must(template.New("").Funcs(funcMap).Parse(serverTemplate)).Execute(file, s)
}

var serverTemplate = `package server

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
{{range $key, $value := .Imports }}
	"{{$value}}"
{{- end }}
)

func New(api {{.API}}, a AuthConfig) *Server {
	s := &Server{
		Auth: a,
		API: api,
	}

	s.registerHandlers()

	return s
}

type AuthConfig map[Endpoint]func(token string) (bool, error)

type Server struct {
	Auth AuthConfig
	API {{.API}}
}

type Endpoint int

var (
{{- range $key, $value := .Paths }}
	{{$value.Camelcase}}Endpoint Endpoint = {{ $key }}
{{- end }}
	AllEndpoints Endpoint = {{(len .Paths)}}
)

func (ep Endpoint) Path() string {
	switch ep {
	case AllEndpoints:
		return "**"
{{- range $key, $value := .Paths }}
	case {{$value.Camelcase}}Endpoint:
		return "/{{$value.Lowercase}}"
{{- end }}
	default:
		return ""
	}
}

func (s *Server) registerHandlers() {
{{- range $key, $value := .Handlers }}
	http.HandleFunc("/{{$value.URL}}", s.Wrap({{$value.Method}}Endpoint, Handle{{$value.Method}}(s.API)))
{{- end }}
}

func (s *Server) Wrap(e Endpoint, fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Kind, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		// Check to see if the 'AllEndpoints' type was set
		authFunc, ok := s.Auth[AllEndpoints]
		if ok {
			allow, msg, reason := checkAuth(w, r, authFunc)
			if !allow {
				http.Error(w, msg, reason)
				return
			}
		} else {
			// Check to see if there is auth setup for this endpoint as there 
			// is no config for all the routes.
			authFunc, ok = s.Auth[e]
			if ok {
				allow, msg, reason := checkAuth(w, r, authFunc)
				if !allow {
					http.Error(w, msg, reason)
					return
				}
			}
		}

		fn(w, r)
	}
}

func checkAuth(w http.ResponseWriter, r *http.Request, authFunc func(token string) (bool, error)) (bool, string, int) {
	t := strings.TrimSpace(r.Header.Get("Authorization"))
	allow, err := authFunc(t)
	if err != nil {
		http.Error(w, "unauthorised", http.StatusUnauthorized)
		return false, "no authorization token present", http.StatusUnauthorized
	}

	return allow, "", 0
}
{{ range $key, $value := .Handlers }}
type {{$value.RequestType}}Request struct {
{{- range $key, $value := $value.Types.Request }}
{{- if eq $value.Type 1}}
	{{$value.Name | ToCamelCase}} {{if ne $value.GoPackage ""}}{{$value.GoPackage}}.{{end}}{{$value.Kind}}
{{- end}}
{{- if eq $value.Type 2}}
	{{$value.Name  | ToCamelCase}} []{{if ne $value.GoPackage ""}}{{$value.GoPackage}}.{{end}}{{$value.Kind}}
{{- end}}
{{- end }}
}

type {{$value.RequestType}}Response struct {
{{- range $key, $value := $value.Types.Response }}
{{- if eq $value.Type 1}}
	{{$value.Name | ToCamelCase}} {{if ne $value.GoPackage ""}}{{$value.GoPackage}}.{{end}}{{$value.Kind}}
{{- end}}
{{- if eq $value.Type 2}}
	{{$value.Name | ToCamelCase}} []{{if ne $value.GoPackage ""}}{{$value.GoPackage}}.{{end}}{{$value.Kind}}
{{- end}}
{{- end }}
}

func Handle{{$value.Method}}(api {{.API}}) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var req {{$value.RequestType}}Request
		err = json.Unmarshal(b, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		{{ range $key2, $value2 := $value.Results }}{{if $key2}}, {{end}}{{ $value2 }}{{ end }}{{ if eq (len $value.Results) 1 }} = {{ end }}{{ if not (eq (len $value.Results) 1) }} := {{ end }}api.{{$value.Method}}(r.Context(){{range $key3, $value3 := .Params }}, req.{{ $value3  | ToCamelCase }}{{end }})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var resp {{$value.ResponseType}}Response
		{{range $key2, $value2 := $value.ResponseParams }}{{if $key2}}, {{end}}{{if eq $value2 "_"}}{{else if $value2}}resp.{{end}}{{$value2  | ToCamelCase }}{{end}} = {{ range $key3, $value3 := .Results }}{{if $key3}}, {{end}}{{ $value3 }}{{ end }}
	
		respBody, err := json.Marshal(resp)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}
		
		w.WriteHeader(http.StatusOK)
		_, err = w.Write(respBody)
		if err != nil {
			_, _ = w.Write([]byte(err.Error()))
			return
		}
	}
}
{{ end }}
`
