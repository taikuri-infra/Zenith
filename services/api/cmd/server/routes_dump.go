package main

import (
	"fmt"
	"io"
	"sort"

	"github.com/gofiber/fiber/v2"
)

// dumpRoutes writes every registered route as "METHOD /path", one per line,
// sorted by path then method. HEAD (auto-added for GET), duplicates, and
// app.Use middleware are omitted. Used by ZENITH_DUMP_ROUTES=1 for audits.
func dumpRoutes(app *fiber.App, w io.Writer) error {
	type route struct{ method, path string }
	seen := map[string]bool{}
	var routes []route
	for _, r := range app.GetRoutes(true) {
		if r.Method == fiber.MethodHead {
			continue
		}
		key := r.Method + " " + r.Path
		if seen[key] {
			continue
		}
		seen[key] = true
		routes = append(routes, route{r.Method, r.Path})
	}
	sort.Slice(routes, func(i, j int) bool {
		if routes[i].path != routes[j].path {
			return routes[i].path < routes[j].path
		}
		return routes[i].method < routes[j].method
	})
	for _, r := range routes {
		if _, err := fmt.Fprintf(w, "%s %s\n", r.method, r.path); err != nil {
			return err
		}
	}
	return nil
}
