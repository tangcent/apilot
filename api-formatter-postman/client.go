package postman

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/tangcent/apilot/api-formatter-postman/model"
)

const defaultPostmanAPIBase = "https://api.getpostman.com"

type PostmanClient struct {
	APIKey  string
	BaseURL string
	HTTP    *http.Client
}

type CreateCollectionRequest struct {
	Collection CollectionWrapper `json:"collection"`
}

type CollectionWrapper struct {
	Info CollectionInfo `json:"info"`
	Item []model.Item   `json:"item"`
}

type CollectionInfo struct {
	Name        string `json:"name"`
	Schema      string `json:"schema,omitempty"`
	Description string `json:"description,omitempty"`
}

type CreateCollectionResponse struct {
	Collection struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		UID  string `json:"uid"`
	} `json:"collection"`
}

type UpdateCollectionResponse struct {
	Collection struct {
		ID   string `json:"id"`
		Name string `json:"name"`
		UID  string `json:"uid"`
	} `json:"collection"`
}

type PostmanAPIError struct {
	StatusCode int
	Name       string `json:"name"`
	Message    string `json:"message"`
}

func (e *PostmanAPIError) Error() string {
	return fmt.Sprintf("postman api error (status %d): %s - %s", e.StatusCode, e.Name, e.Message)
}

func newPostmanClient(apiKey string) PostmanClient {
	return PostmanClient{
		APIKey:  apiKey,
		BaseURL: defaultPostmanAPIBase,
		HTTP:    &http.Client{},
	}
}

func (c PostmanClient) CreateCollection(workspaceID string, col model.Collection) (*CreateCollectionResponse, error) {
	reqBody := CreateCollectionRequest{
		Collection: CollectionWrapper{
			Info: CollectionInfo{
				Name:   col.Info.Name,
				Schema: col.Info.Schema,
			},
			Item: col.Item,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling collection: %w", err)
	}

	url := c.BaseURL + "/collections"
	if workspaceID != "" {
		url = url + "?workspace=" + workspaceID
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		var apiErr PostmanAPIError
		apiErr.StatusCode = resp.StatusCode
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			apiErr.Message = string(respBody)
		}
		if apiErr.Name == "" && apiErr.Message == "" {
			apiErr.Message = fmt.Sprintf("empty response body (%d bytes): %s", len(respBody), string(respBody))
		}
		return nil, &apiErr
	}

	var result CreateCollectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

func (c PostmanClient) UpdateCollection(collectionUID string, col model.Collection) (*UpdateCollectionResponse, error) {
	reqBody := CreateCollectionRequest{
		Collection: CollectionWrapper{
			Info: CollectionInfo{
				Name:   col.Info.Name,
				Schema: col.Info.Schema,
			},
			Item: col.Item,
		},
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshaling collection: %w", err)
	}

	url := c.BaseURL + "/collections/" + collectionUID

	req, err := http.NewRequest("PUT", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr PostmanAPIError
		apiErr.StatusCode = resp.StatusCode
		if err := json.Unmarshal(respBody, &apiErr); err != nil {
			apiErr.Message = string(respBody)
		}
		return nil, &apiErr
	}

	var result UpdateCollectionResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}
	return &result, nil
}

type APIResult struct {
	CollectionID  string `json:"collectionId,omitempty"`
	CollectionUID string `json:"collectionUid,omitempty"`
	CollectionURL string `json:"collectionUrl,omitempty"`
	Action        string `json:"action,omitempty"`
}
