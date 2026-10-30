package llm

import "encoding/json"

// LiteralNewlineDisplay changes only JSON string newline escapes for the model.
// The returned display is not JSON; the outer transport must still json.Marshal it.
func LiteralNewlineDisplay(canonical []byte) (string, error) {
	var value any
	if err := json.Unmarshal(canonical, &value); err != nil {
		return "", err
	}
	result := make([]byte, 0, len(canonical))
	inString := false
	for i := 0; i < len(canonical); i++ {
		c := canonical[i]
		if inString && c == '\\' {
			i++
			if canonical[i] == 'n' {
				result = append(result, '\n')
			} else {
				result = append(result, '\\', canonical[i])
			}
			continue
		}
		if c == '"' {
			inString = !inString
		}
		result = append(result, c)
	}
	return string(result), nil
}
