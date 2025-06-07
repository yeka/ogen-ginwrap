package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"gopkg.in/yaml.v3"
)

type OpenAPISpec struct {
	Paths map[string]map[string]interface{} `yaml:"paths" json:"paths"`
}

type Route struct {
	Path    string
	GinPath string
	Methods []string // uppercased HTTP methods, e.g. GET, POST
}

type TemplateData struct {
	PackageName string
	Routes      []Route
}

func convertPathToGin(path string) string {
	parts := strings.Split(path, "/")
	for i, part := range parts {
		if len(part) == 0 {
			continue
		}
		if part[0] == '{' && part[len(part)-1] == '}' {
			parts[i] = ":" + part[1:len(part)-1]
		}
	}
	return strings.Join(parts, "/")
}

const tpl = `package {{.PackageName}}

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers routes from OpenAPI spec into gin.Engine
// basePath add base path to the gin router. Your ogenHandler must also have prefix to match gin router.
func RegisterRoutes(r *gin.Engine, ogenHandler http.Handler, basePath string) {
	if basePath != "" {
		if basePath[0] != '/' {
			basePath = "/" + basePath
		}
		basePath = strings.TrimRight(basePath, "/")
	}

{{- range .Routes }}
	{{- $path := .GinPath }}
	{{- range .Methods }}
	r.{{ . }}(basePath + "{{ $path }}", gin.WrapH(ogenHandler))
	{{- end }}
{{- end }}
}
`

func main() {
	var (
		inputFile   string
		outputFile  string
		packageName string
	)
	flag.StringVar(&inputFile, "file", "", "Path to OpenAPI YAML or JSON input file")
	flag.StringVar(&outputFile, "out", "", "Output Go filename (optional). If empty, prints to stdout.")
	flag.StringVar(&packageName, "pkg", "main", "Package name for generated Go code")
	flag.Parse()

	if inputFile == "" {
		log.Fatal("You must provide -file argument")
	}

	data, err := os.ReadFile(inputFile)
	if err != nil {
		log.Fatalf("Failed to read input file: %v", err)
	}

	var spec OpenAPISpec
	ext := strings.ToLower(filepath.Ext(inputFile))
	switch ext {
	case ".yaml", ".yml":
		err = yaml.Unmarshal(data, &spec)
	case ".json":
		err = json.Unmarshal(data, &spec)
	default:
		log.Fatalf("Unsupported file extension: %s", ext)
	}
	if err != nil {
		log.Fatalf("Failed to parse OpenAPI spec: %v", err)
	}

	validMethods := map[string]bool{
		"get":     true,
		"post":    true,
		"put":     true,
		"delete":  true,
		"patch":   true,
		"head":    true,
		"options": true,
		"trace":   true,
	}

	routes := []Route{}
	for path, pathItem := range spec.Paths {
		methods := []string{}
		for method := range pathItem {
			methodLower := strings.ToLower(method)
			if validMethods[methodLower] {
				methods = append(methods, strings.ToUpper(methodLower)) // uppercase here
			}
		}
		if len(methods) == 0 {
			continue
		}
		routes = append(routes, Route{
			Path:    path,
			GinPath: convertPathToGin(path),
			Methods: methods,
		})
	}

	tmplData := TemplateData{
		PackageName: packageName,
		Routes:      routes,
	}

	t := template.Must(template.New("routes").Parse(tpl))

	var out *os.File
	if outputFile != "" {
		dir := filepath.Dir(outputFile)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Fatalf("Failed to create output directory: %v", err)
		}
		out, err = os.Create(outputFile)
		if err != nil {
			log.Fatalf("Failed to create output file: %v", err)
		}
		defer out.Close()
	} else {
		out = os.Stdout
	}

	if err := t.Execute(out, tmplData); err != nil {
		log.Fatalf("Failed to execute template: %v", err)
	}
}
