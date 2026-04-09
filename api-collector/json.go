package collector

import "encoding/json"

// MarshalEndpoints serializes a slice of ApiEndpoint to JSON.
func MarshalEndpoints(endpoints []ApiEndpoint) ([]byte, error) {
	return json.Marshal(endpoints)
}

// UnmarshalEndpoints deserializes a JSON byte slice into a slice of ApiEndpoint.
func UnmarshalEndpoints(data []byte) ([]ApiEndpoint, error) {
	var endpoints []ApiEndpoint
	if err := json.Unmarshal(data, &endpoints); err != nil {
		return nil, err
	}
	return endpoints, nil
}
