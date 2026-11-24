package kernel

import (
	"net/http"
	"strings"
)

// Params contient les paramètres de route capturés ({nom}).
type Params map[string]string

type Handler func(w http.ResponseWriter, r *http.Request, p Params)

type route struct {
	method  string
	segs    []string
	names   []string // noms des paramètres {x} dans l'ordre
	handler Handler
}

// Router est un routeur minimal : méthodes + patterns "/a/{b}/c".
type Router struct {
	routes []route
	NotFound Handler
}

func NewRouter() *Router {
	return &Router{NotFound: func(w http.ResponseWriter, r *http.Request, p Params) {
		http.Error(w, "404 - page introuvable", http.StatusNotFound)
	}}
}

func (rt *Router) Handle(method, pattern string, h Handler) {
	segs := splitPath(pattern)
	var names []string
	for _, s := range segs {
		if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
			names = append(names, s[1:len(s)-1])
		}
	}
	rt.routes = append(rt.routes, route{method: method, segs: segs, names: names, handler: h})
}

func (rt *Router) HandleFunc(method, pattern string, h Handler) { rt.Handle(method, pattern, h) }

func splitPath(p string) []string {
	p = strings.Trim(p, "/")
	if p == "" {
		return nil
	}
	return strings.Split(p, "/")
}

func (rt *Router) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathSegs := splitPath(r.URL.Path)
	for _, rt2 := range rt.routes {
		if rt2.method != r.Method {
			continue
		}
		if len(rt2.segs) != len(pathSegs) {
			continue
		}
		params := Params{}
		ok := true
		ni := 0
		for i, seg := range rt2.segs {
			if strings.HasPrefix(seg, "{") && strings.HasSuffix(seg, "}") {
				params[rt2.names[ni]] = pathSegs[i]
				ni++
			} else if seg != pathSegs[i] {
				ok = false
				break
			}
		}
		if ok {
			rt2.handler(w, r, params)
			return
		}
	}
	rt.NotFound(w, r, Params{})
}
