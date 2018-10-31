package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"strings"
	"testing"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func Example() {
	var conn *ssh.Client

	// open an SFTP session over an existing ssh connection.
	sftp, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer sftp.Close()

	// walk a directory
	w := sftp.Walk("/home/user")
	for w.Step() {
		if w.Err() != nil {
			continue
		}
		log.Println(w.Path())
	}

	// leave your mark
	f, err := sftp.Create("hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	if _, err := f.Write([]byte("Hello world!")); err != nil {
		log.Fatal(err)
	}

	// check it's there
	fi, err := sftp.Lstat("hello.txt")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(fi)
}

func TestExampleNewClientPipe(t *testing.T) {
	// Connect to a remote host and request the sftp subsystem via the 'ssh'
	// command.  This assumes that passwordless login is correctly configured.
	cmd := exec.Command("ssh", "root@111.235.181.129", "-p 443", "-s", "sftp")

	// send errors from ssh to stderr
	cmd.Stderr = os.Stderr

	// get stdin and stdout
	pw, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	pr, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	// start the process
	if err := cmd.Start(); err != nil {
		log.Fatal(err)
	}
	defer cmd.Wait()

	// open the SFTP session
	client, err := sftp.NewClientPipe(pr, pw)
	if err != nil {
		log.Fatal(err)
	}

	// read a directory
	list, err := client.ReadDir("/docker")
	if err != nil {
		log.Fatal(err)
	}

	// print contents
	for _, item := range list {
		fmt.Println(item.Name(), item.IsDir())
		if item.IsDir() {
			list, err := client.ReadDir("/docker/" + item.Name())
			if err != nil {
				log.Fatal(err)
			}
			for _, item := range list {
				fmt.Println("\t", item.Name(), item.IsDir())

			}
		}
	}

	// close the connection
	client.Close()
}

func ExampleClient_Mkdir_parents() {
	// Example of mimicing 'mkdir --parents'; I.E. recursively create
	// directoryies and don't error if any directories already exists.
	var conn *ssh.Client

	client, err := sftp.NewClient(conn)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	sshFxFailure := uint32(4)
	mkdirParents := func(client *sftp.Client, dir string) (err error) {
		var parents string

		if path.IsAbs(dir) {
			// Otherwise, an absolute path given below would be turned in to a relative one
			// by splitting on "/"
			parents = "/"
		}

		for _, name := range strings.Split(dir, "/") {
			if name == "" {
				// Paths with double-/ in them should just move along
				// this will also catch the case of the first character being a "/", i.e. an absolute path
				continue
			}
			parents = path.Join(parents, name)
			err = client.Mkdir(parents)
			if status, ok := err.(*sftp.StatusError); ok {
				if status.Code == sshFxFailure {
					var fi os.FileInfo
					fi, err = client.Stat(parents)
					if err == nil {
						if !fi.IsDir() {
							return fmt.Errorf("File exists: %s", parents)
						}
					}
				}
			}
			if err != nil {
				break
			}
		}
		return err
	}

	err = mkdirParents(client, "/tmp/foo/bar")
	if err != nil {
		log.Fatal(err)
	}
}