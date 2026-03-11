package slack

import "encoding/json"

func CompactJSON(data []byte) ([]byte, error) {
	var raw any
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	cleaned := stripEmpty(raw)
	return json.Marshal(cleaned)
}

func stripEmpty(v any) any {
	switch val := v.(type) {
	case map[string]any:
		out := make(map[string]any)
		for k, v := range val {
			cleaned := stripEmpty(v)
			if isEmpty(cleaned) {
				continue
			}
			out[k] = cleaned
		}
		if len(out) == 0 {
			return nil
		}
		return out
	case []any:
		var out []any
		for _, item := range val {
			cleaned := stripEmpty(item)
			out = append(out, cleaned)
		}
		if len(out) == 0 {
			return nil
		}
		return out
	default:
		return val
	}
}

func isEmpty(v any) bool {
	if v == nil {
		return true
	}
	switch val := v.(type) {
	case string:
		return val == ""
	case map[string]any:
		return len(val) == 0
	case []any:
		return len(val) == 0
	case float64:
		return val == 0
	case bool:
		return !val
	default:
		return false
	}
}
