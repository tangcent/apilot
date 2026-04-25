package fastify

import (
	"strings"

	tree_sitter "github.com/tree-sitter/go-tree-sitter"

	"github.com/tangcent/apilot/api-collector-node/express"
	model "github.com/tangcent/apilot/api-model"
)

func AnalyzeFastifyHandler(callNode *tree_sitter.Node, source []byte) *FastifyHandlerInfo {
	info := &FastifyHandlerInfo{}

	handlerNode := findHandlerNode(callNode, source)
	if handlerNode == nil {
		return info
	}

	switch handlerNode.Kind() {
	case "arrow_function":
		info = analyzeArrowHandler(handlerNode, source)
	case "function_expression":
		info = analyzeFunctionHandler(handlerNode, source)
	}

	return info
}

func findHandlerNode(callNode *tree_sitter.Node, source []byte) *tree_sitter.Node {
	argsNode := findChildByKind(callNode, "arguments")
	if argsNode == nil {
		return nil
	}

	pathFound := false
	optionsFound := false

	for i := uint(0); i < argsNode.ChildCount(); i++ {
		child := argsNode.Child(i)
		if child.Kind() == "(" || child.Kind() == ")" || child.Kind() == "," {
			continue
		}

		if !pathFound {
			if child.Kind() == "string" {
				pathFound = true
			}
			continue
		}

		if !optionsFound && child.Kind() == "object" {
			optionsFound = true
			continue
		}

		return child
	}

	return nil
}

func analyzeArrowHandler(node *tree_sitter.Node, source []byte) *FastifyHandlerInfo {
	info := &FastifyHandlerInfo{}
	params := findChildByKind(node, "formal_parameters")
	if params == nil {
		return info
	}

	info.ReqBodyType, info.QueryType, info.ParamsType = extractFastifyRequestTypes(params, source)
	info.ResBodyType = extractFastifyResponseType(params, source)

	return info
}

func analyzeFunctionHandler(node *tree_sitter.Node, source []byte) *FastifyHandlerInfo {
	info := &FastifyHandlerInfo{}
	params := findChildByKind(node, "formal_parameters")
	if params == nil {
		return info
	}

	info.ReqBodyType, info.QueryType, info.ParamsType = extractFastifyRequestTypes(params, source)
	info.ResBodyType = extractFastifyResponseType(params, source)

	return info
}

func extractFastifyRequestTypes(params *tree_sitter.Node, source []byte) (reqBodyType string, queryType string, paramsType string) {
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child.Kind() == "required_parameter" || child.Kind() == "optional_parameter" {
			paramType := extractParamTypeAnnotation(child, source)
			paramName := extractParamName(child, source)

			if paramName == "request" || paramName == "req" {
				if strings.Contains(paramType, "FastifyRequest") {
					reqBodyType, queryType, paramsType = parseFastifyRequestGenerics(paramType)
				}
			}
		}
	}
	return
}

func extractFastifyResponseType(params *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < params.ChildCount(); i++ {
		child := params.Child(i)
		if child.Kind() == "required_parameter" || child.Kind() == "optional_parameter" {
			paramType := extractParamTypeAnnotation(child, source)
			paramName := extractParamName(child, source)

			if paramName == "reply" || paramName == "response" || paramName == "res" {
				if strings.Contains(paramType, "FastifyReply") {
					return parseFastifyReplyGenerics(paramType)
				}
			}
		}
	}
	return ""
}

func parseFastifyRequestGenerics(requestType string) (reqBodyType string, queryType string, paramsType string) {
	if !strings.Contains(requestType, "<") || !strings.Contains(requestType, ">") {
		return "", "", ""
	}

	start := strings.Index(requestType, "<")
	end := strings.LastIndex(requestType, ">")
	if start >= end {
		return "", "", ""
	}

	inner := requestType[start+1 : end]

	if strings.Contains(inner, ":") || strings.Contains(inner, ";") {
		return parseFastifyRequestInlineObject(inner)
	}

	args := splitTypeArgs(inner)
	if len(args) >= 1 {
		paramsType = strings.TrimSpace(args[0])
		if paramsType == "RawServerDefault" || paramsType == "{}" {
			paramsType = ""
		}
	}
	if len(args) >= 2 {
		reqBodyType = strings.TrimSpace(args[1])
		if reqBodyType == "RawBodyDefault" || reqBodyType == "{}" || reqBodyType == "any" {
			reqBodyType = ""
		}
	}
	if len(args) >= 3 {
		queryType = strings.TrimSpace(args[2])
		if queryType == "RawQueryDefault" || queryType == "{}" || queryType == "any" {
			queryType = ""
		}
	}

	return reqBodyType, queryType, paramsType
}

func parseFastifyRequestInlineObject(inner string) (reqBodyType string, queryType string, paramsType string) {
	inner = strings.TrimPrefix(inner, "{")
	inner = strings.TrimSuffix(inner, "}")

	parts := strings.Split(inner, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		kv := strings.SplitN(part, ":", 2)
		if len(kv) != 2 {
			continue
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])

		switch key {
		case "Body":
			if value != "" && value != "any" && value != "{}" {
				reqBodyType = value
			}
		case "Querystring", "Query":
			if value != "" && value != "any" && value != "{}" {
				queryType = value
			}
		case "Params":
			if value != "" && value != "any" && value != "{}" {
				paramsType = value
			}
		}
	}

	return reqBodyType, queryType, paramsType
}

func parseFastifyReplyGenerics(replyType string) string {
	if !strings.Contains(replyType, "<") || !strings.Contains(replyType, ">") {
		return ""
	}

	start := strings.Index(replyType, "<")
	end := strings.LastIndex(replyType, ">")
	if start >= end {
		return ""
	}

	inner := replyType[start+1 : end]
	args := splitTypeArgs(inner)

	if len(args) >= 1 {
		bodyType := strings.TrimSpace(args[0])
		if bodyType != "" && bodyType != "any" && bodyType != "{}" && bodyType != "RawReplyDefaultExpression" {
			return bodyType
		}
	}

	return ""
}

func extractParamTypeAnnotation(paramNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
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

func extractParamName(paramNode *tree_sitter.Node, source []byte) string {
	for i := uint(0); i < paramNode.ChildCount(); i++ {
		child := paramNode.Child(i)
		if child.Kind() == "identifier" {
			return child.Utf8Text(source)
		}
	}
	return ""
}

func findChildByKind(node *tree_sitter.Node, kind string) *tree_sitter.Node {
	for i := uint(0); i < node.ChildCount(); i++ {
		child := node.Child(i)
		if child.Kind() == kind {
			return child
		}
	}
	return nil
}

func splitTypeArgs(s string) []string {
	var args []string
	depth := 0
	start := 0

	for i, c := range s {
		switch c {
		case '<':
			depth++
		case '>':
			depth--
		case ',':
			if depth == 0 {
				arg := strings.TrimSpace(s[start:i])
				if arg != "" {
					args = append(args, arg)
				}
				start = i + 1
			}
		}
	}

	if start < len(s) {
		arg := strings.TrimSpace(s[start:])
		if arg != "" {
			args = append(args, arg)
		}
	}

	return args
}

func ResolveFastifyHandlerTypes(handlerInfo *FastifyHandlerInfo, registry *express.TSTypeRegistry) (reqBody *model.ObjectModel, resBody *model.ObjectModel) {
	resolver := express.NewTSTypeResolver(registry)

	if handlerInfo.ReqBodyType != "" {
		reqBody = resolver.Resolve(handlerInfo.ReqBodyType, nil)
	}

	if handlerInfo.ResBodyType != "" {
		resBody = resolver.Resolve(handlerInfo.ResBodyType, nil)
	}

	return reqBody, resBody
}
