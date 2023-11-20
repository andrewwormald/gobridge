// Code generated by gobridge; DO NOT EDIT.

package server

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"time"
	"github.com/luno/gobridge/example/backend"
	"github.com/luno/gobridge/example/backend/second"
)

func New(api backend.Example, a AuthConfig, basicAuth func(ctx context.Context, token string) (bool, error)) *Server {
	s := &Server{
		AdditionalAuth: a,
		Basic: basicAuth,
		API: api,
	}

	s.registerHandlers()

	return s
}

type AuthConfig map[Endpoint]func(ctx context.Context, token string) (bool, error)

type Server struct {
	AdditionalAuth AuthConfig
	Basic          func(ctx context.Context, token string) (bool, error)
	API backend.Example
}

type Endpoint int

var (
	HasPermissionEndpoint Endpoint = 0
	WhatsTheTimeEndpoint Endpoint = 1
	AllEndpoints Endpoint = 2
)

func (ep Endpoint) Path() string {
	switch ep {
	case AllEndpoints:
		return "**"
	case HasPermissionEndpoint:
		return "/backend/haspermission"
	case WhatsTheTimeEndpoint:
		return "/backend/whatsthetime"
	default:
		return ""
	}
}

func (s *Server) registerHandlers() {
	http.HandleFunc("/backend/haspermission", s.Wrap(HasPermissionEndpoint, HandleHasPermission(s.API)))
	http.HandleFunc("/backend/whatsthetime", s.Wrap(WhatsTheTimeEndpoint, HandleWhatsTheTime(s.API)))
}

func (s *Server) Wrap(e Endpoint, fn func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Kind, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		allow, msg, reason := checkAuth(w, r, s.Basic)
		if !allow {
			http.Error(w, msg, reason)
			return
		}
		
		// Check to see if the 'AllEndpoints' type was set
		authFunc, ok := s.AdditionalAuth[AllEndpoints]
		if ok {
			allow, msg, reason := checkAuth(w, r, authFunc)
			if !allow {
				http.Error(w, msg, reason)
				return
			}
		} else {
			// Check to see if there is auth setup for this endpoint as there 
			// is no config for all the routes.
			authFunc, ok = s.AdditionalAuth[e]
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

func checkAuth(w http.ResponseWriter, r *http.Request, authFunc func(ctx context.Context, token string) (bool, error)) (bool, string, int) {
	t := strings.TrimSpace(r.Header.Get("Authorization"))
	allow, err := authFunc(r.Context(), t)
	if err != nil {
		http.Error(w, "unauthorised", http.StatusUnauthorized)
		return false, "no authorization token present", http.StatusUnauthorized
	}

	if !allow {
		return false, "unauthorised", http.StatusUnauthorized
	}

	return true, "", http.StatusOK
}

type HasPermissionRequest struct {
	R []backend.Role
	U backend.User
	InventoryUpdate map[int64]bool
}

type HasPermissionResponse struct {
	Bool bool
}

func HandleHasPermission(api backend.Example) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var req HasPermissionRequest
		err = json.Unmarshal(b, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		t := strings.TrimSpace(r.Header.Get("Authorization"))
		ctx := context.WithValue(r.Context(), "authorization_header", t)

		uqid, err := api.HasPermission(ctx, req.R, req.U, req.InventoryUpdate)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var resp HasPermissionResponse
		resp.Bool, _ = uqid, err
	
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

type WhatsTheTimeRequest struct {
	Date time.Time
	Toy second.Toy
}

type WhatsTheTimeResponse struct {
	Bool bool
}

func HandleWhatsTheTime(api backend.Example) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var req WhatsTheTimeRequest
		err = json.Unmarshal(b, &req)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		t := strings.TrimSpace(r.Header.Get("Authorization"))
		ctx := context.WithValue(r.Context(), "authorization_header", t)

		epfq, err := api.WhatsTheTime(ctx, req.Date, req.Toy)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(err.Error()))
			return
		}

		var resp WhatsTheTimeResponse
		resp.Bool, _ = epfq, err
	
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

