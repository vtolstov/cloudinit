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
	"path/filepath"
	"strings"
	"unicode"

	"github.com/vtolstov/go-ioctl"
)

func rootMount() (string, error) {
	var err error
	var device string

	f, err := os.Open("/proc/self/mounts")
	if err != nil {
		return device, err
	}
	defer f.Close()
	br := bufio.NewReader(f)

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			break
		}

		fields := strings.Fields(line)
		if fields[1] != "/" || fields[0][0] != '/' {
			continue
		}
		fi, err := os.Stat(fields[0])
		if err != nil {
			return device, err
		}

		if fi.Mode()&os.ModeSymlink == 0 {
			device, err = filepath.EvalSymlinks(fields[0])
			if err != nil {
				return device, err
			}
		} else {
			device = fields[0]
		}
	}
	return device, nil
}

func rootDevice() (string, error) {
	var device string

	mountpoint, err := rootMount()
	if err != nil {
		return device, err
	}

	numstrip := func(r rune) rune {
		if unicode.IsNumber(r) {
			return -1
		}
		return r
	}
	return strings.Map(numstrip, mountpoint), nil
}

func ResizeRootFS() error {
	var err error
	var stdout io.ReadCloser
	var stdin bytes.Buffer

	mountpoint, err := rootMount()
	if err != nil {
		return err
	}

	partstart := "2048"
	device, err := rootDevice()
	if err != nil {
		return err
	}

	mbr := make([]byte, 446)

	f, err := os.OpenFile(device, os.O_RDONLY, os.FileMode(0400))
	if err != nil {
		return err
	}
	_, err = io.ReadFull(f, mbr)
	f.Close()
	if err != nil {
		return err
	}

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

		if strings.HasPrefix(line, mountpoint) {
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
	if err != nil {
		return err
	}
	_, err = w.Write(mbr)
	if err != nil {
		return err
	}

	err = ioctl.BlkRRPart(w.Fd())
	w.Close()
	if err != nil {
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
				if err = exec.Command(name, args...).Run(); err == nil {
					break
				}
			}
		}
	}
	log.Printf("resize filesystem via %s %s", "resize2fs", mountpoint)
	err = exec.Command("resize2fs", mountpoint).Run()
	if err != nil {
		return err
	}
	return nil
}
