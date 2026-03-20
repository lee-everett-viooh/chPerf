package renderer

import (
	"context"
	"net/http"

	"github.com/a-h/templ"
	"github.com/gin-gonic/gin/render"
)

// HTMLTemplRenderer implements Gin's HTMLRender for templ components.
type HTMLTemplRenderer struct {
	Fallback render.HTMLRender
}

// Instance returns a Render for the given data. If data is a templ.Component, renders it.
func (r *HTMLTemplRenderer) Instance(name string, data any) render.Render {
	if c, ok := data.(templ.Component); ok {
		return &TemplRender{Ctx: context.Background(), Component: c}
	}
	if r.Fallback != nil {
		return r.Fallback.Instance(name, data)
	}
	return &TemplRender{Component: nil}
}

// New creates a Render for a templ component with status code.
func New(ctx context.Context, status int, component templ.Component) *TemplRender {
	return &TemplRender{Ctx: ctx, Status: status, Component: component}
}

// TemplRender implements render.Render for templ.
type TemplRender struct {
	Ctx       context.Context
	Status    int
	Component templ.Component
}

// Render writes the component to the response.
func (t *TemplRender) Render(w http.ResponseWriter) error {
	t.WriteContentType(w)
	if t.Status > 0 {
		w.WriteHeader(t.Status)
	}
	if t.Component != nil {
		return t.Component.Render(t.Ctx, w)
	}
	return nil
}

// WriteContentType sets the Content-Type header.
func (t *TemplRender) WriteContentType(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
}
