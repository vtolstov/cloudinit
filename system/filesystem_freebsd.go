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
	"io"
	"log"
	"os/exec"
	"strings"
)

func ResizeRootFS() error {
	var err error
	var stdout io.ReadCloser
	var device string
	var partition string

	cmd := exec.Command("mount", "-t", "ufs", "-p")
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to get mounted file systems %s\n", err.Error())
		return err
	}
	r := bufio.NewReader(stdout)

	if err = cmd.Start(); err != nil {
		log.Printf("failed to get mounted file systems %s\n", err.Error())
		return err
	}

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}

		ps := strings.Fields(line) // /dev/da0s1a             /                       ufs     rw              1 1
		if ps[1] == "/" {
			var i int
			if i = strings.Index(ps[0], "s"); i < 0 {
				return fmt.Errorf("failed to find slice number")
			}
			device = ps[0][:i]
			partition = strings.TrimSuffix(ps[0][i:], "a")
		}
	}

	if err = cmd.Wait(); err != nil || partition == "" {
		return fmt.Errorf("failed to find partition on %s\n", device)
	}

	commands := []string{
		"sysctl kern.geom.debugflags=16",
		"gpart resize -i 1 " + strings.Split(device, "/")[2],
		"gpart resize -i 1 " + strings.Split(device, "/")[2] + partition,
		"true > " + device,
		"true > " + device + partition,
		"true > " + device + partition + "a",
		"gpart resize -i 1 " + strings.Split(device, "/")[2],
		"gpart resize -i 1 " + strings.Split(device, "/")[2] + partition,
		"growfs -y /",
	}

	for _, command := range commands {
		log.Printf("resize fs %s\n", command)
		exec.Command(strings.Split(command, " ")[0], strings.Split(command, " ")[1:]...).Run()
	}
	return nil
}
