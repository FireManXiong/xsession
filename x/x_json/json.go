package x_json

import "encoding/json"

// 序列化
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// 反序列化
func Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
