package vshttp

import (
	"sort"
	"strings"
)

type Router struct {
	routes      map[routeKey]Handler
	paramRoutes []paramRoute
}

type routeKey struct {
	method Method
	path   string
}

type paramRoute struct {
	method  Method
	pattern string
	parts   []routePart
	handler Handler
}

type routePart struct {
	value   string
	isParam bool
}

func NewRouter() *Router {
	return &Router{
		routes: make(map[routeKey]Handler),
	}
}

func (r *Router) Handle(method Method, path string, handler Handler) {
	validateRoute(method, path, handler)

	if isParamPath(path) {
		r.paramRoutes = append(r.paramRoutes, paramRoute{
			method:  method,
			pattern: path,
			parts:   parseRouteParts(path),
			handler: handler,
		})
		return
	}

	r.routes[routeKey{method: method, path: path}] = handler
}

func (r *Router) Get(path string, handler ContextHandlerFunc) {
	r.Handle(GET, path, handler)
}

func (r *Router) Post(path string, handler ContextHandlerFunc) {
	r.Handle(POST, path, handler)
}

func (r *Router) Put(path string, handler ContextHandlerFunc) {
	r.Handle(PUT, path, handler)
}

func (r *Router) Patch(path string, handler ContextHandlerFunc) {
	r.Handle(PATCH, path, handler)
}

func (r *Router) Delete(path string, handler ContextHandlerFunc) {
	r.Handle(DELETE, path, handler)
}

func (r *Router) ServeHTTP(w ResponseWriter, req *Request) {
	key := routeKey{method: req.Method, path: req.Path}
	if handler, ok := r.routes[key]; ok {
		handler.ServeHTTP(w, req)
		return
	}

	if handler, params, ok := r.matchParams(req.Method, req.Path); ok {
		req.Params = params
		handler.ServeHTTP(w, req)
		return
	}

	if r.pathExistsForOtherMethod(req.Method, req.Path) {
		w.Header().Replace("Allow", r.allowedMethods(req.Path))
		_ = w.Text(MethodNotAllowed, "method not allowed")
		return
	}

	_ = w.Text(NotFound, "not found")
}

func (r *Router) matchParams(method Method, path string) (Handler, map[string]string, bool) {
	pathParts := splitPath(path)
	for _, route := range r.paramRoutes {
		if route.method != method || len(route.parts) != len(pathParts) {
			continue
		}

		params := make(map[string]string)
		matched := true
		for i, part := range route.parts {
			if part.isParam {
				params[part.value] = pathParts[i]
				continue
			}
			if part.value != pathParts[i] {
				matched = false
				break
			}
		}
		if matched {
			return route.handler, params, true
		}
	}
	return nil, nil, false
}

func (r *Router) pathExistsForOtherMethod(method Method, path string) bool {
	for key := range r.routes {
		if key.path == path && key.method != method {
			return true
		}
	}

	for _, route := range r.paramRoutes {
		if route.method == method {
			continue
		}
		if _, _, ok := (&Router{paramRoutes: []paramRoute{route}}).matchParams(route.method, path); ok {
			return true
		}
	}
	return false
}

func (r *Router) allowedMethods(path string) string {
	methods := make([]string, 0)
	seen := make(map[Method]bool)
	for key := range r.routes {
		if key.path == path && !seen[key.method] {
			methods = append(methods, key.method.String())
			seen[key.method] = true
		}
	}
	for _, route := range r.paramRoutes {
		if seen[route.method] {
			continue
		}
		if _, _, ok := (&Router{paramRoutes: []paramRoute{route}}).matchParams(route.method, path); ok {
			methods = append(methods, route.method.String())
			seen[route.method] = true
		}
	}
	sort.Strings(methods)
	return strings.Join(methods, ", ")
}

func validateRoute(method Method, path string, handler Handler) {
	if !method.Valid() {
		panic("invalid HTTP method: " + method.String())
	}
	if handler == nil {
		panic("handler must not be nil")
	}
	if !strings.HasPrefix(path, "/") {
		panic("route path must start with /")
	}
}

func isParamPath(path string) bool {
	for _, part := range splitPath(path) {
		if isParamPart(part) {
			return true
		}
	}
	return false
}

func parseRouteParts(path string) []routePart {
	pathParts := splitPath(path)
	parts := make([]routePart, 0, len(pathParts))
	for _, part := range pathParts {
		if isParamPart(part) {
			if paramName(part) == "" {
				panic("route parameter name must not be empty")
			}
			parts = append(parts, routePart{
				value:   paramName(part),
				isParam: true,
			})
			continue
		}
		parts = append(parts, routePart{value: part})
	}
	return parts
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return []string{}
	}
	return strings.Split(trimmed, "/")
}

func isParamPart(part string) bool {
	return strings.HasPrefix(part, ":") || strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}")
}

func paramName(part string) string {
	if strings.HasPrefix(part, ":") {
		return strings.TrimPrefix(part, ":")
	}
	return strings.TrimSuffix(strings.TrimPrefix(part, "{"), "}")
}
