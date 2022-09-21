package clis

func interfacesToString(list []interface{}) []string {
	if list == nil {
		return nil
	}

	result := make([]string, len(list))
	for i, item := range list {
		if item == nil {
			result[i] = ""
		} else {
			result[i] = item.(string)
		}
	}

	return result
}

func unique(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}
