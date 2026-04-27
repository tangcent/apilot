package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	formatter "github.com/tangcent/apilot/api-formatter"
)

type FileCollectionStore struct{}

var _ formatter.CollectionStore = (*FileCollectionStore)(nil)

func NewCollectionStore() *FileCollectionStore {
	return &FileCollectionStore{}
}

func (s *FileCollectionStore) GetBinding(project string) (*formatter.CollectionBinding, error) {
	bindings, err := loadCollectionBindings()
	if err != nil {
		return nil, err
	}
	if b, ok := bindings[project]; ok {
		return &formatter.CollectionBinding{
			WorkspaceID:   b.WorkspaceID,
			CollectionUID: b.CollectionUID,
		}, nil
	}
	return nil, nil
}

func (s *FileCollectionStore) SetBinding(project string, binding formatter.CollectionBinding) error {
	bindingsMu.Lock()
	defer bindingsMu.Unlock()
	bindings, err := loadCollectionBindings()
	if err != nil {
		return err
	}
	bindings[project] = collectionEntry{
		WorkspaceID:   binding.WorkspaceID,
		CollectionUID: binding.CollectionUID,
	}
	return saveCollectionBindings(bindings)
}

type collectionEntry struct {
	WorkspaceID   string `json:"workspaceId,omitempty"`
	CollectionUID string `json:"collectionUid,omitempty"`
}

type collectionBindings map[string]collectionEntry

var bindingsMu sync.Mutex

func CollectionsFilePath() string {
	dir := ConfigDir()
	if dir == "" {
		return ""
	}
	return filepath.Join(dir, "postman_collections.json")
}

func loadCollectionBindings() (collectionBindings, error) {
	path := CollectionsFilePath()
	if path == "" {
		return collectionBindings{}, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return collectionBindings{}, nil
		}
		return nil, err
	}
	var bindings collectionBindings
	if err := json.Unmarshal(data, &bindings); err != nil {
		return collectionBindings{}, nil
	}
	if bindings == nil {
		bindings = collectionBindings{}
	}
	return bindings, nil
}

func saveCollectionBindings(bindings collectionBindings) error {
	path := CollectionsFilePath()
	if path == "" {
		return fmt.Errorf("cannot determine config path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(bindings, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func RemoveCollectionBinding(project string) error {
	bindingsMu.Lock()
	defer bindingsMu.Unlock()
	bindings, err := loadCollectionBindings()
	if err != nil {
		return err
	}
	delete(bindings, project)
	return saveCollectionBindings(bindings)
}

func ListCollectionBindings() (map[string]formatter.CollectionBinding, error) {
	entries, err := loadCollectionBindings()
	if err != nil {
		return nil, err
	}
	result := make(map[string]formatter.CollectionBinding, len(entries))
	for k, v := range entries {
		result[k] = formatter.CollectionBinding{
			WorkspaceID:   v.WorkspaceID,
			CollectionUID: v.CollectionUID,
		}
	}
	return result, nil
}
