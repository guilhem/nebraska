// Package codegen provides primitives to interact with the openapi HTTP API.
//
// Code generated by github.com/deepmap/oapi-codegen version v1.8.2 DO NOT EDIT.
package codegen

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// Base64 encoded, gzipped, json marshaled Swagger object
var swaggerSpec = []string{

	"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",
}

// GetSwagger returns the content of the embedded swagger specification file
// or error if failed to decode
func decodeSpec() ([]byte, error) {
	zipped, err := base64.StdEncoding.DecodeString(strings.Join(swaggerSpec, ""))
	if err != nil {
		return nil, fmt.Errorf("error base64 decoding spec: %s", err)
	}
	zr, err := gzip.NewReader(bytes.NewReader(zipped))
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}
	var buf bytes.Buffer
	_, err = buf.ReadFrom(zr)
	if err != nil {
		return nil, fmt.Errorf("error decompressing spec: %s", err)
	}

	return buf.Bytes(), nil
}

var rawSpec = decodeSpecCached()

// a naive cached of a decoded swagger spec
func decodeSpecCached() func() ([]byte, error) {
	data, err := decodeSpec()
	return func() ([]byte, error) {
		return data, err
	}
}

// Constructs a synthetic filesystem for resolving external references when loading openapi specifications.
func PathToRawSpec(pathToFile string) map[string]func() ([]byte, error) {
	var res = make(map[string]func() ([]byte, error))
	if len(pathToFile) > 0 {
		res[pathToFile] = rawSpec
	}

	return res
}

// GetSwagger returns the Swagger specification corresponding to the generated code
// in this file. The external references of Swagger specification are resolved.
// The logic of resolving external references is tightly connected to "import-mapping" feature.
// Externally referenced files must be embedded in the corresponding golang packages.
// Urls can be supported but this task was out of the scope.
func GetSwagger() (swagger *openapi3.T, err error) {
	var resolvePath = PathToRawSpec("")

	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	loader.ReadFromURIFunc = func(loader *openapi3.Loader, url *url.URL) ([]byte, error) {
		var pathToFile = url.String()
		pathToFile = path.Clean(pathToFile)
		getSpec, ok := resolvePath[pathToFile]
		if !ok {
			err1 := fmt.Errorf("path not found: %s", pathToFile)
			return nil, err1
		}
		return getSpec()
	}
	var specData []byte
	specData, err = rawSpec()
	if err != nil {
		return
	}
	swagger, err = loader.LoadFromData(specData)
	if err != nil {
		return
	}
	return
}