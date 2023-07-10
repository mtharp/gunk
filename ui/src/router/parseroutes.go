package router

import (
	_ "embed"
	"regexp"
)

//go:embed index.ts
var router string

func IndexRoutes() (routes []string) {
	routeRe := regexp.MustCompile(`\Wpath: ['"](.*?)['"]`)
	paramRe := regexp.MustCompile(`:([^/]+)`)
	for _, m := range routeRe.FindAllStringSubmatch(router, -1) {
		route := m[1]
		route = paramRe.ReplaceAllString(route, "{$1}")
		routes = append(routes, route)
	}
	return
}
