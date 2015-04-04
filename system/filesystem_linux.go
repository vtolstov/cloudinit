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
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/vtolstov/go-ioctl"
)

func ResizeRootFS() error {
	var err error
	var stdout io.ReadCloser
	var stdin bytes.Buffer

	output, err := exec.Command("findmnt", "-n", "-o", "source", "/").CombinedOutput()
	if err != nil {
		return err
	}

	mountpoint := strings.TrimSpace(string(output))
	partstart := "2048"
	device := mountpoint[:len(mountpoint)-1]
	partition := mountpoint[len(mountpoint)-1:]

	cmd := exec.Command("fdisk", "-l", "-u", device)
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		log.Printf("failed to open %s via fdisk %s 2\n", device, err.Error())
		return err
	}
	r := bufio.NewReader(stdout)

	if err = cmd.Start(); err != nil {
		log.Printf("failed to open %s via fdisk %s 3\n", device, err.Error())
		return err
	}

	for {
		line, err := r.ReadString('\n')
		if err != nil {
			break
		}

		if strings.HasPrefix(line, device+partition) {
			ps := strings.Fields(line) // /dev/sda1      *      4096   251658239   125827072  83 Linux
			if ps[1] == "*" {
				partstart = ps[2]
			} else {
				partstart = ps[1]
			}
		}
	}

	if err = cmd.Wait(); err != nil || partstart == "" {
		return fmt.Errorf("failed to open %s via fdisk 4\n", device)
	}

	stdin.Write([]byte("o\nn\np\n1\n" + partstart + "\n\na\n1\nw\n"))
	cmd = exec.Command("fdisk", "-u", device)
	cmd.Stdin = &stdin
	cmd.Run()
	stdin.Reset()

	w, err := os.OpenFile(device, os.O_WRONLY, 0600)
	if err == nil {
		defer w.Close()
		err = ioctl.BlkRRPart(w.Fd())
		if err == nil {
			return exec.Command("resize2fs", device+partition).Run()
		}
	}

	args := []string{}
	for _, name := range []string{"partx", "partprobe", "kpartx"} {
		if _, err = exec.LookPath(name); err == nil {
			switch name {
			case "partx":
				args = []string{"-u", device}
			default:
				args = []string{device}
			}
			log.Printf("update partition table via %s %s", name, strings.Join(args, " "))
			exec.Command(name, args...).Run()
		}
	}
	log.Printf("resize filesystem via %s %s", "resize2fs", device+partition)
	err = exec.Command("resize2fs", device+partition).Run()
	if err != nil {
		return err
	}
	for _, name := range []string{"grub-install", "grub2-install"} {
		if _, err = exec.LookPath(name); err == nil {
			log.Printf("reinstall grub %s %s", name, device)
			exec.Command(name, device).Run()
		}
	}
}
