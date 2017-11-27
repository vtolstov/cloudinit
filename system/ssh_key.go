// Copyright 2015 CoreOS, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package system

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

func diffLines(src, dst []string) []string {
	var tgt []string

	mb := map[string]bool{}

	for _, x := range src {
		mb[x] = true
	}

	for _, x := range dst {
		if _, ok := mb[x]; !ok {
			mb[x] = true
		}
	}

	for k, _ := range mb {
		tgt = append(tgt, k)
	}

	return tgt
}

func readLines(path string) ([]string, error) {
	var lines []string

	file, err := os.Open(path)
	if err != nil {
		return lines, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// writeLines writes the lines to the given file.
func writeLines(lines []string, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	w := bufio.NewWriter(file)
	for _, line := range lines {
		fmt.Fprintln(w, line)
	}
	return w.Flush()
}

// Add the provide SSH public key to the core user's list of
// authorized keys
func AuthorizeSSHKeys(user string, keysName string, keys []string) error {
	for name, key := range keys {
		keys[name] = strings.TrimSpace(key)
	}

	// join all keys with newlines, ensuring the resulting string
	// also ends with a newline
	// joined := fmt.Sprintf("%s\n", strings.Join(keys, "\n"))

	home, err := UserHome(user)
	if err != nil {
		return err
	}

	if _, err = os.Stat(home + "/.ssh"); err != nil {
		if err = os.MkdirAll(home+"/.ssh", os.FileMode(0755)); err != nil {
			return err
		}
	}

	authorized_file := fmt.Sprintf("%s/.ssh/authorized_keys", home)
	var newkeys []string
	for _, x := range keys {
		newkeys = append(newkeys, strings.Split(x, "\n")...)
	}
	oldkeys, _ := readLines(authorized_file)

	diffkeys := diffLines(oldkeys, newkeys)
	return writeLines(diffkeys, authorized_file)
}
