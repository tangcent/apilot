package formatter

import "encoding/json"

// MarshalOptions serializes FormatOptions to JSON.
func MarshalOptions(opts FormatOptions) ([]byte, error) {
	return json.Marshal(opts)
}

// UnmarshalOptions deserializes JSON into FormatOptions.
func UnmarshalOptions(data []byte) (FormatOptions, error) {
	var opts FormatOptions
	if err := json.Unmarshal(data, &opts); err != nil {
		return FormatOptions{}, err
	}
	return opts, nil
}
