package nestjs

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-node/express"
	model "github.com/tangcent/apilot/api-model"
)

func AnalyzeNestJSHandler(methodNode *tree_sitter.Node, source []byte) *NestJSHandlerInfo {
	info := &NestJSHandlerInfo{}

	paramsNode := findChildByKindResolver(methodNode, "formal_parameters")
	if paramsNode != nil {
		info.BodyType = extractBodyType(paramsNode, source)
	}

	info.ReturnType = extractReturnType(methodNode, source)

	info.ApiResponses = extractApiResponses(methodNode, source)

	return info
}

func extractBodyType(paramsNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramsNode.ChildCount(); i++ {
		child := paramsNode.Child(i)
		if child.Kind() != "required_parameter" && child.Kind() != "optional_parameter" {
			continue
		}

		hasBodyDecorator := false
		for j := uint(0); j < child.ChildCount(); j++ {
			paramChild := child.Child(j)
			if paramChild.Kind() == "decorator" {
				decoratorName := extractNestJSDecoratorName(paramChild, source)
				if decoratorName == "Body" {
					hasBodyDecorator = true
					break
				}
			}
		}

		if hasBodyDecorator {
			return extractParamTypeText(child, source)
		}
	}

	return ""
}

func extractReturnType(methodNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < methodNode.ChildCount(); i++ {
		child := methodNode.Child(i)
		if child.Kind() == "type_annotation" {
			return extractTypeAnnotationText(child, source)
		}
	}
	return ""
}

func extractTypeAnnotationText(node *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == ":" {
			continue
		}
		return child.Utf8Text(source)
	}
	return ""
}

func extractParamTypeText(paramNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
		if child.Kind() == "type_annotation" {
			return extractTypeAnnotationText(child, source)
		}
	}
	return ""
}

func extractNestJSDecoratorName(decoratorNode *tree_sitter.Node, source []byte) string {
	callNode := findChildByKindResolver(decoratorNode, "call_expression")
	if callNode != nil {
		for i := uint(0); i < callNode.ChildCount(); i++ {
			child := callNode.Child(i)
			if child.Kind() == "identifier" {
				return child.Utf8Text(source)
			}
			if child.Kind() == "member_expression" {
				for j := uint(0); j < child.ChildCount(); j++ {
					memberChild := child.Child(j)
					if memberChild.Kind() == "property_identifier" {
						return memberChild.Utf8Text(source)
					}
				}
			}
		}
	}

	for i := uint(0); i < decoratorNode.ChildCount(); i++ {
		child := decoratorNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}

	return ""
}

func extractApiResponses(methodNode *tree_sitter.Node, source []byte) []ApiResponseInfo {
	var responses []ApiResponseInfo

	prev := methodNode.PrevNamedSibling()
	for prev != nil && prev.Kind() == "decorator" {
		resp := parseApiResponseDecorator(prev, source)
		if resp != nil {
			responses = append(responses, *resp)
		}
		prev = prev.PrevNamedSibling()
	}

	return responses
}

func parseApiResponseDecorator(decoratorNode *tree_sitter.Node, source []byte) *ApiResponseInfo {
	callNode := findChildByKindResolver(decoratorNode, "call_expression")
	if callNode == nil {
		return nil
	}

	decoratorName := extractNestJSDecoratorName(decoratorNode, source)
	if decoratorName != "ApiResponse" && decoratorName != "ApiOkResponse" &&
		decoratorName != "ApiCreatedResponse" && decoratorName != "ApiBadRequestResponse" {
		return nil
	}

	resp := &ApiResponseInfo{}

	argsNode := findChildByKindResolver(callNode, "arguments")
	if argsNode == nil {
		return resp
	}

	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "object" {
			resp = extractApiResponseObject(child, source, resp)
			break
		}
	}

	return resp
}

func extractApiResponseObject(objNode *tree_sitter.Node, source []byte, resp *ApiResponseInfo) *ApiResponseInfo {
	for i := uint(0); i < objNode.ChildCount(); i++ {
		child := objNode.Child(i)
		if child.Kind() != "pair" {
			continue
		}

		key := extractResolverPairKey(child, source)
		switch key {
		case "status":
			val := extractResolverPairStringValue(child, source)
			resp.Status = val
		case "description":
			val := extractResolverPairStringValue(child, source)
			resp.Description = val
		case "type":
			val := extractResolverPairStringValue(child, source)
			if val != "" {
				resp.TypeName = val
			}
		case "isArray":
			for j := uint(0); j < child.ChildCount(); j++ {
				pairChild := child.Child(j)
				if pairChild.Kind() == "true" {
					resp.IsArray = true
				}
			}
		}
	}

	return resp
}

func ResolveNestJSHandlerTypes(handlerInfo *NestJSHandlerInfo, registry *express.TSTypeRegistry) (reqBody *model.ObjectModel, resBody *model.ObjectModel) {
	return ResolveNestJSHandlerTypesWithDepResolver(handlerInfo, registry, nil)
}

func ResolveNestJSHandlerTypesWithDepResolver(handlerInfo *NestJSHandlerInfo, registry *express.TSTypeRegistry, depResolver collector.DependencyResolver) (reqBody *model.ObjectModel, resBody *model.ObjectModel) {
	resolver := express.NewTSTypeResolver(registry)
	if depResolver != nil {
		resolver.SetDependencyResolver(depResolver)
	}

	if handlerInfo.BodyType != "" {
		bodyType := unwrapPromise(handlerInfo.BodyType)
		reqBody = resolver.Resolve(bodyType, nil)
	}

	returnType := handlerInfo.ReturnType
	if returnType != "" {
		returnType = unwrapPromise(returnType)
		resBody = resolver.Resolve(returnType, nil)
	}

	if len(handlerInfo.ApiResponses) > 0 {
		resp := pickBestApiResponse(handlerInfo.ApiResponses)
		if resp != nil && resp.TypeName != "" {
			apiRespType := resp.TypeName
			if resp.IsArray {
				apiRespType = "Array<" + apiRespType + ">"
			}
			resolved := resolver.Resolve(apiRespType, nil)
			if resolved != nil && !resolved.IsNull() {
				if resBody == nil || resBody.IsNull() {
					resBody = resolved
				}
			}
		}
	}

	return reqBody, resBody
}

func pickBestApiResponse(responses []ApiResponseInfo) *ApiResponseInfo {
	for _, resp := range responses {
		if resp.Status == "200" || resp.Status == "201" {
			return &resp
		}
	}

	for _, resp := range responses {
		if resp.TypeName != "" {
			return &resp
		}
	}

	return nil
}

func unwrapPromise(typeStr string) string {
	typeStr = strings.TrimSpace(typeStr)
	if strings.HasPrefix(typeStr, "Promise<") && strings.HasSuffix(typeStr, ">") {
		inner := typeStr[len("Promise<") : len(typeStr)-1]
		return strings.TrimSpace(inner)
	}
	return typeStr
}

func findChildByKindResolver(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func extractResolverPairKey(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "property_identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "string" {
			return unquoteResolverString(child.Utf8Text(source))
		}
	}
	return ""
}

func extractResolverPairStringValue(pairNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < pairNode.ChildCount(); i++ {
		child := pairNode.Child(i)
		if child.Kind() == "string" {
			return unquoteResolverString(child.Utf8Text(source))
		}
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
		if child.Kind() == "number" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func unquoteResolverString(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' {
		return s[1 : len(s)-1]
	}
	return s
}
