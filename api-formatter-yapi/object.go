package yapi

import (
	"bytes"
	"encoding/json"
)

// object is a JSON object that keeps its keys in insertion order. YApi lists
// properties in the order the schema gives them, so the order the source
// declares fields in has to survive serialisation; a Go map would sort them.
type object struct {
	keys []string
	vals map[string]any
}

func newObject() *object { return &object{vals: map[string]any{}} }

func (o *object) set(key string, value any) {
	if _, ok := o.vals[key]; !ok {
		o.keys = append(o.keys, key)
	}
	o.vals[key] = value
}

func (o *object) has(key string) bool {
	_, ok := o.vals[key]
	return ok
}

func (o *object) get(key string) any { return o.vals[key] }

func (o *object) MarshalJSON() ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteByte('{')
	for i, key := range o.keys {
		if i > 0 {
			buf.WriteByte(',')
		}
		k, err := json.Marshal(key)
		if err != nil {
			return nil, err
		}
		v, err := json.Marshal(o.vals[key])
		if err != nil {
			return nil, err
		}
		buf.Write(k)
		buf.WriteByte(':')
		buf.Write(v)
	}
	buf.WriteByte('}')
	return buf.Bytes(), nil
}
