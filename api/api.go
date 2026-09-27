// Package api хранит OpenAPI-спецификацию сервиса и страницу документации Scalar.
// Файлы вшиты в бинарник и отдаются роутами /openapi.yaml и /docs.
package api

import _ "embed"

var (
	// Spec — OpenAPI-спецификация (openapi.yaml).
	//go:embed openapi.yaml
	Spec []byte

	// DocsHTML — страница Scalar, читающая спецификацию с /openapi.yaml.
	//go:embed docs.html
	DocsHTML []byte
)
