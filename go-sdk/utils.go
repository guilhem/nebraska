package nebraska

import "github.com/kinvolk/nebraska/backend/pkg/codegen"

func convertReqEditors(reqEditors ...RequestEditorFn) []codegen.RequestEditorFn {
	var codegenReqEditors []codegen.RequestEditorFn
	for _, reqEditor := range reqEditors {
		codegenReqEditors = append(codegenReqEditors, codegen.RequestEditorFn(reqEditor))
	}
	return codegenReqEditors
}
