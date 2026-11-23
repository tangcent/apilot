package javacollector

import (
	"testing"

	collector "github.com/tangcent/apilot/api-collector"
	"github.com/tangcent/apilot/api-collector-java/maven"
	"github.com/tangcent/apilot/api-collector-java/parser"
	"github.com/tangcent/apilot/api-collector-java/springmvc"
)

// supertypeLibrary are the classes of a library that the project extends and
// implements: a generic class, a generic controller and an interface.
var supertypeLibrary = map[string]string{
	"lib.Parent": `package lib;

public class Parent<T> {
    private String inherited;
    private T payload;
    private Item ownItem;
}
`,
	"lib.Item": `package lib;

public class Item {
    private String libraryItem;
}
`,
	"lib.Contract": `package lib;

public interface Contract {
    String KEY = "k";
}
`,
	"lib.BaseController": `package lib;

import org.springframework.web.bind.annotation.GetMapping;
import org.springframework.web.bind.annotation.RequestMapping;

public abstract class BaseController<T> {
    @GetMapping("/one")
    public T one() {
        return null;
    }
}
`,
}

// A superclass or an interface written with its qualified name is read, and one
// that only a dependency declares is read from it: the fields it declares are the
// subclass's, the arguments the extends clause gives it are the classes that clause
// names, and the endpoints of a generic controller of a library are the
// controller's own.
func TestASuperclassWrittenInFullIsReadFromTheProjectAndFromTheDependencies(t *testing.T) {
	dir := t.TempDir()
	writeTree(t, dir, map[string]string{
		"src/main/java/common/ProjectParent.java": "package common;\n\npublic class ProjectParent {\n    private String inheritedFromProject;\n}\n",
		"src/main/java/app/dto/Item.java":         "package app.dto;\n\npublic class Item {\n    private String projectItem;\n}\n",
		"src/main/java/app/dto/Child.java":        "package app.dto;\n\npublic class Child extends lib.Parent<Item> implements lib.Contract {\n    private String own;\n}\n",
		"src/main/java/app/dto/Plain.java":        "package app.dto;\n\npublic class Plain extends common.ProjectParent {\n}\n",
		"src/main/java/app/web/ItemController.java": `package app.web;

import app.dto.Child;
import app.dto.Item;
import app.dto.Plain;
import org.springframework.web.bind.annotation.*;

@RestController
public class ItemController extends lib.BaseController<Item> {

    @GetMapping("/child")
    public Child child() {
        return null;
    }

    @GetMapping("/plain")
    public Plain plain() {
        return null;
    }
}
`,
	})
	p, err := parser.NewParser(parser.ParserOptions{})
	if err != nil {
		t.Fatal(err)
	}
	defer p.Close()
	results, err := p.ParseDirectory(dir)
	if err != nil {
		t.Fatal(err)
	}

	var asked []string
	index, err := maven.NewMavenDependencyResolver()
	if err != nil {
		t.Fatal(err)
	}
	defer index.Close()
	index.SetRunner(scriptedIndex(supertypeLibrary, &asked))

	sm := springmvc.NewParser()
	sm.SetDependencyResolver(index)
	unresolved := collector.NewUnresolvedSet()
	sm.SetUnresolved(unresolved)
	endpoints := map[string]collector.ApiEndpoint{}
	for _, ctrl := range sm.ExtractControllers(results) {
		for _, ep := range ctrl.Endpoints {
			endpoints[ep.MethodName] = springmvcEndpointToAPI(ep, ctrl.Name)
		}
	}

	for name, want := range map[string]string{
		// lib.Parent<Item> is the library's class with the project's Item as its
		// argument; its own Item is the library's.
		"child": `{inherited:string,own:string,ownItem:{libraryItem:string},payload:{projectItem:string}}`,
		"plain": `{inheritedFromProject:string}`,
		// The endpoint of the library's controller, typed by its parameter.
		"one": `{projectItem:string}`,
	} {
		ep, ok := endpoints[name]
		if !ok || ep.Response == nil {
			t.Errorf("endpoint %s has no response: %+v", name, ep)
			continue
		}
		if got := nestedShape(ep.Response.Body); got != want {
			t.Errorf("%s:\n got  %s\n want %s", name, got, want)
		}
	}
	if got := unresolved.Counts(); len(got) != 0 {
		t.Errorf("unresolved = %v, want none", got)
	}
	// Each library class is asked of the index once.
	seen := map[string]int{}
	for _, name := range asked {
		seen[name]++
	}
	for name, n := range seen {
		if n != 1 {
			t.Errorf("%s was asked of the index %d times: %v", name, n, asked)
		}
	}
	if seen["lib.Parent"] != 1 || seen["lib.BaseController"] != 1 {
		t.Errorf("asked = %v, want lib.Parent and lib.BaseController once each", asked)
	}
}
