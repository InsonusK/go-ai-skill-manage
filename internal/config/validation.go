package config

func linkFolders(s map[string]any) ([]string, error) {
	current := s
	for _, key := range []string{"validation", "rules", "link"} {
		raw, exists := current[key]
		if !exists || raw == nil {
			return []string{"examples"}, nil
		}
		var err error
		current, err = mapping(raw, key)
		if err != nil {
			return nil, err
		}
	}
	raw, exists := current["skip_folder"]
	if !exists {
		return []string{"examples"}, nil
	}
	return listValue(raw, "skip_folder", nil)
}
