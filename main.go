package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/gliderlabs/ssh"
	"github.com/google/shlex"
)

var GIT_RECEIVE_PACK_CMD = "git-receive-pack"
var GIT_UPLOAD_PACK_CMD = "git-upload-pack"

type SshOut struct {
	s ssh.Session
}

type SshIn struct {
	s ssh.Session
}

func (writer SshOut) Write(data []byte) (int, error) {
	return writer.s.Write(data)
}

func (reader SshIn) Read(data []byte) (int, error) {
	return reader.s.Read(data)
}

func runGitReceivePackCmd(s ssh.Session, repoPath string) (string, error) {
	cmd := exec.Command(GIT_RECEIVE_PACK_CMD, repoPath)
	buf := bytes.NewBuffer([]byte{})

	multiOut := io.MultiWriter(buf, SshOut{s: s})

	cmd.Stdout = multiOut
	cmd.Stderr = s.Stderr()
	cmd.Stdin = SshIn{s: s}

	if err := cmd.Run(); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func runGitUploadPackCmd(s ssh.Session, repoPath string) error {
	cmd := exec.Command(GIT_UPLOAD_PACK_CMD, repoPath)

	cmd.Stdout = SshOut{s: s}
	cmd.Stderr = s.Stderr()
	cmd.Stdin = SshIn{s: s}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		fmt.Println(s.RawCommand())
		args, err := shlex.Split(s.RawCommand())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}

		if args[0] == GIT_RECEIVE_PACK_CMD {
			if stdout, err := runGitReceivePackCmd(s, args[1]); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			} else {
				fmt.Println(stdout)
			}
		} else if args[0] == GIT_UPLOAD_PACK_CMD {
			if err := runGitUploadPackCmd(s, args[1]); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil, ssh.HostKeyFile(".ssh/id_ed25519")))
}
