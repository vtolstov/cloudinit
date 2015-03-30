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
	"fmt"
	"os"
	"strings"
)

// Add the provide SSH public key to the core user's list of
// authorized keys
func AuthorizeSSHKeys(user string, keysName string, keys []string) error {
	for name, key := range keys {
		keys[name] = strings.TrimSpace(key)
	}

	// join all keys with newlines, ensuring the resulting string
	// also ends with a newline
	joined := fmt.Sprintf("%s\n", strings.Join(keys, "\n"))

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
	f, err := os.OpenFile(authorized_file, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(joined)

	return err
}
