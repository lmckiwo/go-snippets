package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const (
	DefaultSshTcpTimeout = 10 * time.Second // the default time to establish a connection with SSH.
)

//Error definition
var (
	InvalidHostName = errors.New("invalid parameters: hostname is empty")
	InvalidPort     = errors.New("invalid parameters: port must be range 0 ~ 65535")
)

//Returns the current user name
func getCurrentUser() string {
	user, _ := user.Current()
	return user.Username
}

//Store uploaded or downloaded information
type TransferInfo struct {
	Kind         string // upload or download
	Local        string // local path
	Dst          string // target path
	TransferByte int64  // number of bytes transferred (bytes)
}

func (t *TransferInfo) String() string {
	return fmt.Sprintf(`TransforInfo(Kind:"%s", Local: "%s", Dst: "%s", TransferByte: %d)`,
		t.Kind, t.Local, t.Dst, t.TransferByte)
}

//Structure information for storing execution results
type ExecInfo struct {
	Cmd      string
	Output   []byte
	ExitCode int
}

func (e *ExecInfo) OutputString() string {
	return string(e.Output)
}

func (e *ExecInfo) String() string {
	return fmt.Sprintf(`ExecInfo(cmd: "%s", exitcode: %d)`,
		e.Cmd, e.ExitCode)
}

type AuthConfig struct {
	*ssh.ClientConfig
	User     string
	Password string
	KeyFile  string
	Timeout  time.Duration
}

func (a *AuthConfig) setDefault() {
	if a.User == "" {
		a.User = getCurrentUser()
	}

	if a.KeyFile == "" {
		userHome, _ := os.UserHomeDir()
		a.KeyFile = fmt.Sprintf("%s/.ssh/id_rsa", userHome)
	}

	if a.Timeout == 0 {
		a.Timeout = DefaultSshTcpTimeout
	}
}

func (a *AuthConfig) SetAuthMethod() (ssh.AuthMethod, error) {
	a.setDefault()
	if a.Password != "" {
		return ssh.Password(a.Password), nil
	}
	data, err := ioutil.ReadFile(a.KeyFile)
	if err != nil {
		return nil, err
	}
	singer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, err
	}
	return ssh.PublicKeys(singer), nil
}

func (a *AuthConfig) ApplyConfig() error {
	authMethod, err := a.SetAuthMethod()
	if err != nil {
		return err
	}
	a.ClientConfig = &ssh.ClientConfig{
		User:            a.User,
		Auth:            []ssh.AuthMethod{authMethod},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         a.Timeout,
	}
	return nil
}

//Storing connected structures
type conn struct {
	client     *ssh.Client
	sftpClient *sftp.Client
}

func (c *conn) Close() {
	if c.sftpClient != nil {
		c.sftpClient.Close()
		c.sftpClient = nil
	}
	if c.client != nil {
		c.client.Close()
		c.client = nil
	}
}

//Sshclient structure
type SSHClient struct {
	conn
	HostName   string
	Port       int
	AuthConfig AuthConfig
}

//Set default port information
func (s *SSHClient) setDefaultValue() {
	if s.Port == 0 {
		s.Port = 22
	}
}

//Connect to remote host
func (s *SSHClient) Connect() error {
	if s.client != nil {
		log.Println("Already Login")
		return nil
	}
	if err := s.AuthConfig.ApplyConfig(); err != nil {
		return err
	}
	s.setDefaultValue()
	addr := fmt.Sprintf("%s:%d", s.HostName, s.Port)
	var err error
	s.client, err = ssh.Dial("tcp", addr, s.AuthConfig.ClientConfig)
	if err != nil {
		return err
	}
	return nil
}

//A session can execute commands only once, that is, s.session.combinedoutput cannot be executed multiple times in the same session
//If you want to execute multiple times, you need to create a session for each command (this is done here)
func (s *SSHClient) Exec(cmd string) (*ExecInfo, error) {
	session, err := s.client.NewSession()
	if err != nil {
		return nil, err
	}
	defer session.Close()
	output, err := session.CombinedOutput(cmd)
	var exitcode int
	if err != nil {
		//Convert the assertion to a concrete implementation type and get the return value
		exitcode = err.(*ssh.ExitError).ExitStatus()
	}
	return &ExecInfo{
		Cmd:      cmd,
		Output:   output,
		ExitCode: exitcode,
	}, nil
}

//Upload local files to remote host
func (s *SSHClient) Upload(localPath string, dstPath string) (*TransferInfo, error) {
	transferInfo := &TransferInfo{Kind: "upload", Local: localPath, Dst: dstPath, TransferByte: 0}
	var err error
	//If the SFTP Client is not opened, it will be opened for reuse
	if s.sftpClient == nil {
		if s.sftpClient, err = sftp.NewClient(s.client); err != nil {
			return transferInfo, err
		}
	}
	localFileObj, err := os.Open(localPath)
	if err != nil {
		return transferInfo, err
	}
	defer localFileObj.Close()

	dstFileObj, err := s.sftpClient.Create(dstPath)
	if err != nil {
		return transferInfo, err
	}
	defer dstFileObj.Close()

	written, err := io.Copy(dstFileObj, localFileObj)
	if err != nil {
		return transferInfo, err
	}
	transferInfo.TransferByte = written
	return transferInfo, nil
}

//Download files from remote host to local
func (s *SSHClient) Download(dstPath string, localPath string) (*TransferInfo, error) {
	transferInfo := &TransferInfo{Kind: "download", Local: localPath, Dst: dstPath, TransferByte: 0}
	var err error
	if s.sftpClient == nil {
		if s.sftpClient, err = sftp.NewClient(s.client); err != nil {
			return transferInfo, err
		}
	}
	//defer s.sftpClient.Close()
	localFileObj, err := os.Create(localPath)
	if err != nil {
		return transferInfo, err
	}
	defer localFileObj.Close()

	dstFileObj, err := s.sftpClient.Open(dstPath)
	if err != nil {
		return transferInfo, err
	}
	defer dstFileObj.Close()

	written, err := io.Copy(localFileObj, dstFileObj)
	if err != nil {
		return transferInfo, err
	}
	transferInfo.TransferByte = written
	return transferInfo, nil
}

//Construction method of sshclient
func NewSSHClient(hostname string, port int, authConfig AuthConfig) (*SSHClient, error) {
	switch {
	case hostname == "":
		return nil, InvalidHostName
	case port > 65535 || port < 0:
		return nil, InvalidPort
	}
	sshClient := &SSHClient{HostName: hostname, Port: port, AuthConfig: authConfig}
	err := sshClient.Connect()
	if err != nil {
		return nil, err
	}
	return sshClient, nil
}

func runCommand(wg *sync.WaitGroup, ssh *SSHClient, cmd string) error {
	defer wg.Done()

	fmt.Println("executing: ", cmd)
	execinfo, err := ssh.Exec(cmd)
	fmt.Println(execinfo.OutputString(), err)
	fmt.Printf("executing: %s - done\n", cmd)
	return err
}
func main() {
	var wg sync.WaitGroup
	commands := []string{"ls -l", "/usr/sbin/ifconfig -a", "df -h"}
	//Testing
	sshClient, err := NewSSHClient("192.168.1.2", 22, AuthConfig{User: "username", Password: "password"})
	if err != nil {
		fmt.Println(err)
		return
	}
	defer sshClient.Close()
	//Execute the command for the first time
	//execinfo, err := sshClient.Exec("ls -l")
	//fmt.Println(execinfo.OutputString(), err)
	for _, v := range commands {
		wg.Add(1)
		go runCommand(&wg, sshClient, v)
	}
	//fmt.Println(a)

	//runCommand(sshClient, "sleep 5")

	//Second execution command
	//out1, exitcode2 := sshClient.Exec("/usr/sbin/ifconfig -a")
	////fmt.Println(string(out1), exitcode2)
	//fmt.Println(out1.OutputString(), exitcode2)
	//wg.Add(1)
	//go runCommand(&wg, sshClient, "/usr/sbin/ifconfig -a")
	//fmt.Println(a)

	//Upload file
	transInfoUpload, err := sshClient.Upload("./passwd", "/tmp/password_upload")
	fmt.Println(transInfoUpload, err)
	//Download File
	transInfoDownload, err := sshClient.Download("/tmp/password_upload", "./passwd_download")
	fmt.Println(transInfoDownload, err)

	//wg.Wait()
	fmt.Println("completed")

}
