package nestjs

type NestJSHandlerInfo struct {
	BodyType     string
	ReturnType   string
	ApiResponses []ApiResponseInfo
}

type ApiResponseInfo struct {
	Status      string
	Description string
	TypeName    string
	IsArray     bool
}
