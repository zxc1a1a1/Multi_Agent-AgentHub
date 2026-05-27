package config

import "os"

// ExpandEnvVars expands ${VAR_NAME} with values from the current process
// environment. Missing variables are kept unchanged.
func ExpandEnvVars(input []byte) []byte {
	return expandEnvVars(input, os.LookupEnv)
}

func expandEnvVars(input []byte, lookup func(string) (string, bool)) []byte {
	if len(input) == 0 {
		return nil
	}
	if lookup == nil {
		out := make([]byte, len(input))
		copy(out, input)
		return out
	}

	out := make([]byte, 0, len(input))
	for i := 0; i < len(input); {
		if input[i] == '$' && i+2 < len(input) && input[i+1] == '{' {
			end := i + 2
			for end < len(input) && input[end] != '}' {
				end++
			}
			if end < len(input) {
				key := input[i+2 : end]
				if isValidEnvName(key) {
					if value, ok := lookup(string(key)); ok {
						out = append(out, value...)
					} else {
						out = append(out, input[i:end+1]...)
					}
					i = end + 1
					continue
				}
			}
		}
		out = append(out, input[i])
		i++
	}
	return out
}

func isValidEnvName(name []byte) bool {
	if len(name) == 0 {
		return false
	}
	if !isEnvNameStart(name[0]) {
		return false
	}
	for i := 1; i < len(name); i++ {
		if !isEnvNamePart(name[i]) {
			return false
		}
	}
	return true
}

func isEnvNameStart(ch byte) bool {
	return ch == '_' || (ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z')
}

func isEnvNamePart(ch byte) bool {
	return isEnvNameStart(ch) || (ch >= '0' && ch <= '9')
}
