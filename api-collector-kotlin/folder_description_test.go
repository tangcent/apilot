package kotlincollector

import "testing"

func TestFolderDescriptionIsTheControllerKDoc(t *testing.T) {
	endpoints, _ := collectDir(t, "testdata/folder-description")
	if got := findEndpoint(t, endpoints, "list").FolderDescription; got != "Order management." {
		t.Errorf("list folder description = %q", got)
	}
	if got := findEndpoint(t, endpoints, "ping").FolderDescription; got != "" {
		t.Errorf("a controller without KDoc has no folder description, got %q", got)
	}
}
