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
	"sr+CAAA=",
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