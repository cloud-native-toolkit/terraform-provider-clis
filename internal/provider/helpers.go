// Copyright (c) 2025 Cloud-Native Toolkit
// SPDX-License-Identifier: MIT

package provider

import (
	"bufio"
	"errors"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"log"
	"os"
	"regexp"
)

var armArch = regexp.MustCompile(`^arm`)
var macos = regexp.MustCompile(`darwin`)

type EnvContext struct {
	Arch   string
	Os     string
	Alpine bool
}

func (c EnvContext) isArmArch() bool {
	return armArch.MatchString(c.Arch)
}

func (c EnvContext) isMacOs() bool {
	return macos.MatchString(c.Os)
}

func (c EnvContext) isAlpine() bool {
	return c.Alpine
}

func listTypeToStrings(list types.List) []string {

	// Create a slice of strings to hold the values
	stringArray := make([]string, 0)

	// Append the values into the slice
	for _, v := range list.Elements() {
		if !v.IsNull() && !v.IsUnknown() {
			s, ok := v.(types.String)
			if ok {
				stringArray = append(stringArray, s.ValueString())
			}
		}
	}

	return stringArray
}

func typeStringsToStrings(vals ...types.String) []string {
	stringArray := make([]string, len(vals))

	for i, val := range vals {
		stringArray[i] = val.ValueString()
	}

	return stringArray
}

func first(values ...string) string {

	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}

	return ""
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

func fileExists(filename string) (bool, error) {
	_, err := os.Lstat(filename)
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
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
