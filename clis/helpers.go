package clis

import (
	"bufio"
	"log"
	"os"
)

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

func checkForAlpine() bool {
	if exists, err := fileExists("/etc/os-release"); !exists || err != nil {
		return false
	}

	file, err := os.Open("/etc/os-release")
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if tmpError := file.Close(); tmpError != nil {
			log.Fatal(err)
		}
	}()

	alpine := false
	lines := bufio.NewScanner(file)
	for lines.Scan() {
		line := lines.Text()

		if line == "ID=alpine" {
			alpine = true
			break
		}
	}

	if err := lines.Err(); err != nil {
		log.Fatal(err)
	}

	return alpine
}
