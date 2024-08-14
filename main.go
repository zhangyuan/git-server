package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

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

func runSshCmd(session ssh.Session, command string, repoPath string) error {
	cmd := exec.Command(command, repoPath)
	sshOut := SshOut{s: session}

	cmd.Stdout = sshOut
	cmd.Stderr = session.Stderr()
	cmd.Stdin = SshIn{session: session}

	return cmd.Run()
}

type GitServer struct {
	Addr       string
	HostKeyPEM []byte
	KeysStore  KeysStore

	sshServer *ssh.Server
}

type KeysStore interface {
	Authenticate(key ssh.PublicKey) (*User, error)
}

type KeysFileStore struct {
	ClientKeysPath string
}

type User struct {
}

func (keysStore *KeysFileStore) Authenticate(clientPublicKey ssh.PublicKey) (*User, error) {
	clientKeysBytes, err := os.ReadFile(keysStore.ClientKeysPath)
	if err != nil {
		return nil, err
	}
	rawClientKeys := strings.Split(string(clientKeysBytes), "\n")

	for idx := range rawClientKeys {
		keyStr := rawClientKeys[idx]
		if len(strings.TrimSpace(keyStr)) == 0 {
			continue
		}
		publicKey, _, _, _, err := ssh.ParseAuthorizedKey([]byte(keyStr))
		if err != nil {
			return nil, err
		}
		if ssh.KeysEqual(clientPublicKey, publicKey) {
			return &User{}, nil
		}
	}
	return nil, nil
}

func (server *GitServer) Serve() error {
	sshServer := &ssh.Server{
		Addr:             server.Addr,
		Handler:          server.SshSessionHandler,
		PublicKeyHandler: server.AuthKeyHandler,
	}

	if err := sshServer.SetOption(ssh.HostKeyPEM(server.HostKeyPEM)); err != nil {
		return err
	}

	return sshServer.ListenAndServe()
}

func (server *GitServer) AuthKeyHandler(ctx ssh.Context, key ssh.PublicKey) bool {
	if user, err := server.KeysStore.Authenticate(key); err != nil {
		fmt.Fprintf(os.Stderr, "Authenticate error: %v\n", err)
		return false
	} else if user == nil {
		return false
	}
	return true
}

func (server *GitServer) SshSessionHandler(s ssh.Session) {
	args, err := shlex.Split(s.RawCommand())
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return
	}

	if args[0] == GIT_RECEIVE_PACK_CMD || args[0] == GIT_UPLOAD_PACK_CMD {
		if err := runSshCmd(s, args[0], args[1]); err != nil {
			fmt.Fprintln(os.Stderr, err.Error())
		}
	}
}

func NewGitServer(addr, hostKeyFilePath, clientKeysPath string) (*GitServer, error) {
	hostKeyPEM, err := os.ReadFile(hostKeyFilePath)
	if err != nil {
		return nil, err
	}

	keyStore := &KeysFileStore{
		ClientKeysPath: clientKeysPath,
	}

	return &GitServer{
		Addr:       ":2222",
		KeysStore:  keyStore,
		HostKeyPEM: hostKeyPEM,
	}, nil
}

func main() {
	server, err := NewGitServer(
		":2222",
		".ssh/host_key",
		".ssh/client_keys",
	)
	if err != nil {
		log.Fatalln(err)
	}

	log.Fatal(server.Serve())
}
