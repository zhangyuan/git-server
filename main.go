package main

import (
	"fmt"
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
	session ssh.Session
}

func (writer SshOut) Write(data []byte) (int, error) {
	return writer.s.Write(data)
}

func (reader SshIn) Read(data []byte) (int, error) {
	return reader.session.Read(data)
}

func runGitReceivePackCmd(session ssh.Session, repoPath string) error {
	cmd := exec.Command(GIT_RECEIVE_PACK_CMD, repoPath)
	sshOut := SshOut{s: session}

	cmd.Stdout = sshOut
	cmd.Stderr = session.Stderr()
	cmd.Stdin = SshIn{session: session}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func runGitUploadPackCmd(session ssh.Session, repoPath string) error {
	cmd := exec.Command(GIT_UPLOAD_PACK_CMD, repoPath)
	sshOut := SshOut{s: session}

	cmd.Stdout = sshOut
	cmd.Stderr = session.Stderr()
	cmd.Stdin = SshIn{session: session}

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		fmt.Println("ssh raw command:", s.RawCommand())

		args, err := shlex.Split(s.RawCommand())
		if err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
			return
		}

		if args[0] == GIT_RECEIVE_PACK_CMD {
			if err := runGitReceivePackCmd(s, args[1]); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		} else if args[0] == GIT_UPLOAD_PACK_CMD {
			if err := runGitUploadPackCmd(s, args[1]); err != nil {
				fmt.Fprintln(os.Stderr, err.Error())
			}
		}
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil, ssh.HostKeyFile(".ssh/id_ed25519")))
}
