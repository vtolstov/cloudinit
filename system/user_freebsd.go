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
	"log"
	"os/exec"
	"strings"

	"github.com/coreos/coreos-cloudinit/config"
)

func CreateUser(u *config.User) error {
	args := []string{}

	args = append(args, "useradd")

	if u.Name == "root" {
		args = append(args, "-b", "/")
	} else {
		args = append(args, "-b", "/home")
	}

	args = append(args, "-w", "no")

	if u.GECOS != "" {
		args = append(args, "-c", fmt.Sprintf("%q", u.GECOS))
	}

	if u.Homedir != "" {
		args = append(args, "-d", u.Homedir)
	}

	if !u.NoCreateHome {
		args = append(args, "-m")
	}

	if u.PrimaryGroup != "" {
		args = append(args, "-g", u.PrimaryGroup)
	}

	if len(u.Groups) > 0 {
		args = append(args, "-G", strings.Join(u.Groups, ","))
	}

	if u.Shell != "" {
		args = append(args, "-s", u.Shell)
	}

	args = append(args, "-n", u.Name)

	output, err := exec.Command("pw", args...).CombinedOutput()
	if err != nil {
		log.Printf("Command 'pw useradd %s' failed: %v\n%s", strings.Join(args, " "), err, output)
		return err
	}

	return nil
}

func LockUnlockUser(u *config.User) error {
	args := []string{}

	output, err := exec.Command("getent", "passwd", u.Name).CombinedOutput()
	if err != nil {
		return err
	}
	passwd := strings.Split(string(output), ":")

	if u.LockPasswd {
		if strings.HasPrefix(passwd[1], "*LOCKED*") {
			return nil
		}
		args = append(args, "lock")
	} else {
		if !strings.HasPrefix(passwd[1], "*LOCKED*") {
			return nil
		}
		args = append(args, "unlock")
	}

	args = append(args, u.Name)

	output, err = exec.Command("pw", args...).CombinedOutput()
	if err != nil {
		log.Printf("Command 'pw %s' failed: %v\n%s", strings.Join(args, " "), err, output)
	}
	return err
}

func SetUserPassword(user, hash string) error {
	cmd := exec.Command("pw", "usermod", user, "-H", "0")

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}

	err = cmd.Start()
	if err != nil {
		log.Fatal(err)
	}

	_, err = stdin.Write([]byte(hash))
	if err != nil {
		return err
	}
	stdin.Close()

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
