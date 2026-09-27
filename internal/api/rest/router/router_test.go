package router

import (
	"net/http"
	"strings"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/require"

	"teriyaki-sauce-service/api"
	"teriyaki-sauce-service/internal/api/rest/handler"
)

// Роуты самой документации в спецификации не описываются.
var _docsRoutes = map[string]bool{"/openapi.yaml": true, "/docs": true}

// Спецификация валидна и описывает ровно те роуты, что зарегистрированы в роутере:
// падает, если добавили роут и забыли openapi.yaml или наоборот.
func TestOpenAPIMatchesRouter(t *testing.T) {
	loader := openapi3.NewLoader()
	doc, err := loader.LoadFromData(api.Spec)
	require.NoError(t, err)
	require.NoError(t, doc.Validate(loader.Context))

	var specRoutes []string
	for path, item := range doc.Paths.Map() {
		for method := range item.Operations() {
			specRoutes = append(specRoutes, method+" "+path)
		}
	}

	passthrough := func(next http.Handler) http.Handler { return next }
	r := New(passthrough, &handler.TagHandler{}, &handler.TaskHandler{}, &handler.NoteHandler{})

	var routerRoutes []string
	err = chi.Walk(r.(chi.Routes), func(method, route string, _ http.Handler, _ ...func(http.Handler) http.Handler) error {
		if !_docsRoutes[route] {
			routerRoutes = append(routerRoutes, method+" "+strings.TrimSuffix(route, "/"))
		}

		return nil
	})
	require.NoError(t, err)

	require.ElementsMatch(t, specRoutes, routerRoutes)
}
