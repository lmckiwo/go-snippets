package main

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/ssh"
)

const (
	host = "192.168.1.2"
	port = "22"
)

func main() {
	//Testing
	sshConfig := &ssh.ClientConfig{
		User: "username",
		Auth: []ssh.AuthMethod{
			ssh.Password("password"),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := ssh.Dial("tcp", host+":"+port, sshConfig)
	if err != nil {
		fmt.Println("Failed to dial", err)
        return
	}

	session, err := connection.NewSession()
	if err != nil {
		fmt.Println("Failed to create session:", err)
        return
	}

    defer session.Close()

	// create pty
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4 kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed =14.4 kbaud
	}

	if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
		fmt.Println("request for pseudo terminal failed:")
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		fmt.Println("Unable to setup stdin for session:", err)
        return
	}
	go io.Copy(stdin, os.Stdin)

	stdout, err := session.StdoutPipe()
	if err != nil {
		fmt.Println("Unable to setup stdout for session:")
	}
	go io.Copy(os.Stdout, stdout)

	stderr, err := session.StderrPipe()
	if err != nil {
		fmt.Println("Unable to setup stderr for session:")
	}
	go io.Copy(os.Stderr, stderr)

	err = session.Run("ls -l")
    if err != nil {
        fmt.Println("unable to run command: ", err)
        return
    }
	// err = session.Run("date")
 //    if err != nil {
 //        fmt.Println("unable to run command: ", err)
 //        return
 //    }
	//wg.Wait()
	fmt.Println("completed")

}
