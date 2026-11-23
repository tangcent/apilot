package kotlincollector

import (
	"github.com/tangcent/apilot/api-collector-java/maven"
	javaparser "github.com/tangcent/apilot/api-collector-java/parser"
	ktparser "github.com/tangcent/apilot/api-collector-kotlin/parser"
	"github.com/tangcent/apilot/api-collector-kotlin/resolver"
)

// A class fetched from a library written in Kotlin comes back as Kotlin
// source. Registering the parser lets the dependency resolver read it, for the
// Kotlin collector and the Java collector alike, without the Java module
// importing this one.
func init() {
	maven.RegisterSourceParser("kotlin", parseDependencySource)
}

// parseDependencySource reads a Kotlin dependency source into the Java model
// the same way the project's own declarations are read.
func parseDependencySource(name string, source []byte) ([]javaparser.Class, error) {
	p, err := ktparser.NewParser()
	if err != nil {
		return nil, err
	}
	defer p.Close()

	result, err := p.ParseSource(name, source)
	if err != nil {
		return nil, err
	}
	if result.Error != nil {
		return nil, result.Error
	}
	return resolver.ToJavaClasses(result.Classes, nil, result.TypeAliases), nil
}
