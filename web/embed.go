package web

import "embed"

//go:embed static
var StaticFS embed.FS

//go:embed templates
var TemplateFS embed.FS
