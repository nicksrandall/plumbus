package plumbus

import (
	"fmt"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/jargv/plumbus/generate"
)

type documenter interface {
	Documentation() string
}

type Documentation struct {
	Endpoints    []*Endpoint      `json:"endpoints"`
	Types        map[string]*Type `json:"types,omitempty"`
	Introduction []string
}

type Endpoint struct {
	Method       string               `json:"method,omitempty"`
	Path         string               `json:"path"`
	Description  string               `json:"description,omitempty"`
	RequestBody  string               `json:"requestBody,omitempty"`
	ResponseBody string               `json:"responseBody,omitempty"`
	Params       map[string]ParamInfo `json:"params,omitempty"`
	Notes        []string             `json:"notes,omitempty"`
}

type Type struct {
	Description string      `json:"description,omitempty"`
	Example     interface{} `json:"example"`
}

type ParamInfo struct {
	Type        string `json:"type"`
	Required    bool   `json:"required"`
	Description string `json:"description,omitempty"`
}

func (sm *ServeMux) Documentation(introduction ...string) *Documentation {
	for i, line := range introduction {
		introduction[i] = cleanupText(line)
	}
	d := &Documentation{
		Types:        map[string]*Type{},
		Introduction: introduction,
	}
	d.collectEndpoints(sm.Paths)
	return d
}

func (d *Documentation) collectEndpoints(paths *Paths) {
	for path, segment := range paths.flatten() {
		docs := cleanupText(strings.Join(segment.documentation, "\n"))
		d.collectEndpoint(path, segment.originalHandler, docs)
	}
}

func (d *Documentation) collectEndpoint(path string, handler interface{}, docs string) {
	switch val := handler.(type) {
	case http.HandlerFunc, func(http.ResponseWriter, *http.Request):
		d.Endpoints = append(d.Endpoints, &Endpoint{
			Path:        path,
			Description: docs,
		})

	case ByMethod:
		d.collectMethodEndpoints(path, &val, docs)

	case *ByMethod:
		d.collectMethodEndpoints(path, val, docs)

	default:
		e := d.handlerFunctionToEndpoint(handler)
		e.Path = path
		e.Description = docs
		d.Endpoints = append(d.Endpoints, e)
	}
}

func (d *Documentation) collectMethodEndpoints(path string, handlers *ByMethod, docs string) {
	addHandler := func(method string, handler interface{}) {
		if handler != nil {
			e := d.handlerFunctionToEndpoint(handler)
			e.Method = method
			e.Path = path
			e.Description = docs
			d.Endpoints = append(d.Endpoints, e)
		}
	}

	addHandler("GET", handlers.GET)
	addHandler("POST", handlers.POST)
	addHandler("PUT", handlers.PUT)
	addHandler("PATCH", handlers.PATCH)
	addHandler("DELETE", handlers.DELETE)
	addHandler("OPTIONS", handlers.OPTIONS)
}

func (d *Documentation) handlerFunctionToEndpoint(handler interface{}) *Endpoint {
	typ := reflect.TypeOf(handler)
	if typ.Kind() != reflect.Func {
		return &Endpoint{}
	}

	info, err := generate.CollectInfo(typ)
	if err != nil {
		panic(fmt.Errorf("error generating documentation: %v", err))
	}

	e := &Endpoint{}

	for _, input := range info.Inputs {
		switch t := input.ConversionType; t {
		case generate.ConvertBody:
			e.RequestBody = d.mkType(input.Type)
		case generate.ConvertCustom:
			val := reflect.Zero(input.Type).Interface()
			if doc, ok := val.(documenter); ok {
				e.Notes = append(e.Notes, cleanupText(doc.Documentation()))
			}
		case generate.ConvertIntQueryParam, generate.ConvertStringQueryParam:
			p := ParamInfo{
				Required: input.Type.Kind() != reflect.Ptr,
			}

			val := reflect.Zero(input.Type).Interface()
			if doc, ok := val.(documenter); ok {
				p.Description = cleanupText(doc.Documentation())
			}

			if t == generate.ConvertIntQueryParam {
				p.Type = "string"
			} else {
				p.Type = "integer"
			}

			if e.Params == nil {
				e.Params = map[string]ParamInfo{}
			}

			e.Params[input.Name] = p
		default:
			log.Fatalf("unexpected conversion type %s", t)
		}
	}

	for _, output := range info.Outputs {
		switch t := output.ConversionType; t {
		case generate.ConvertBody:
			e.ResponseBody = d.mkType(output.Type)
		case generate.ConvertCustom:
			val := reflect.Zero(output.Type).Interface()
			if doc, ok := val.(documenter); ok {
				e.Notes = append(e.Notes, cleanupText(doc.Documentation()))
			}
		case generate.ConvertError:
			//not much we can do here
		default:
			log.Fatalf("unexpected conversion type %s", t)
		}
	}

	return e
}

func (d *Documentation) mkType(typ reflect.Type) string {
	name := typeName(typ)

	log.Printf("name: %#v", name)

	if _, ok := d.Types[name]; !ok {
		example := deepZero(typ).Interface()
		description := ""
		if documenter, ok := example.(documenter); ok {
			description = cleanupText(documenter.Documentation())
		}
		d.Types[name] = &Type{
			Example:     example,
			Description: description,
		}
	}

	return name
}

func typeName(typ reflect.Type) string {
	name := fmt.Sprintf("%v", typ)
	parts := strings.Split(name, ".")
	last := parts[len(parts)-1]
	return strings.TrimLeft(last, "*")
}

func deepZero(typ reflect.Type) reflect.Value {
	needsExample := func(v reflect.Value) bool {
		isPtrOrSliceOrMap := v.Kind() == reflect.Ptr || v.Kind() == reflect.Slice || v.Kind() == reflect.Map
		canSet := isPtrOrSliceOrMap && v.CanSet()
		return isPtrOrSliceOrMap && canSet
	}

	if typ.Kind() == reflect.Ptr {
		val := deepZero(typ.Elem())
		if val.CanAddr() {
			return val.Addr()
		} else {
			return reflect.New(typ.Elem())
		}
	}

	if typ.Kind() == reflect.Slice {
		slice := reflect.MakeSlice(typ, 1, 1)
		val := slice.Index(0)
		if needsExample(val) {
			val.Set(deepZero(typ.Elem()))
		}
		return slice
	}

	if typ.Kind() == reflect.Map {
		m := reflect.MakeMap(typ)
		key := deepZero(typ.Key())
		val := deepZero(typ.Elem())
		m.SetMapIndex(key, val)
		return m
	}

	val := reflect.New(typ).Elem()

	if typ.Kind() != reflect.Struct {
		return val
	}

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if needsExample(field) {
			val.Field(i).Set(deepZero(val.Field(i).Type()))
		}
	}
	return val
}

func cleanupText(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimSpace(line)
	}
	return strings.TrimSpace(strings.Join(lines, " "))
}
