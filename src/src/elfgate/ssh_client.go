package elfgate


import(
    "fmt"
    "time"
    "bytes"
    "strings"
    "io/ioutil"

    "github.com/pkg/sftp"
    "golang.org/x/crypto/ssh"
)


type AgentPool struct {
    user     string

    clients  []*SSHClient
}

func NewAgentPool(user, passwd string, hosts []string, o chan *CmdOutput) *AgentPool {
    var clients  []*SSHClient
    var authType string

    if passwd == "" {
        authType = "publickey"
    } else {
        authType = "password"
    }

    for _, host := range hosts {
        if sshClient, err := NewSSHClient(authType, user, passwd, host, o); err != nil {
            o<- &CmdOutput{
                Host   : host,
                Output : []string{err.Error()},
                Error  : err,
            }
        } else {
            clients = append(clients, sshClient)
        }
    }
    return &AgentPool{
        user    : user,
        clients : clients,
    }
}

// Is there alive clients.
func (this *AgentPool) Active() bool {
    if len(this.clients) == 0 {
        return false
    } else {
        return true
    }
}

// Return Alive client len.
func (this *AgentPool) Len() int {
    return len(this.clients)
}

// Exec cmds
func (this *AgentPool) Exec(cmd string, t int) error {
    cmdType    := CmdType(cmd)
    if cmdType == "sftp" {
        fs, dest, err := SftpCmdProc(this.user, Cmd)
        if err != nil {
            return err
        }

        for _, sshClient := range this.clients {
            // sftpFiles    := make([]*SftpFile, len(fs))
            // copy(sftpFiles, fs)             // copy one file list to sshClient
            // go sshClient.RunSftp(dest, sftpFiles)
            go sshClient.RunSftp(dest, fs)
        }
    } else {
        for _, sshClient := range this.clients {
            go sshClient.RunCmd(cmdType, cmd, t)
        }
    }
    return nil
}

// Stop running cmds
func (this *AgentPool) StopCmds() int {
    for _, sshClient := range this.clients {
        go sshClient.StopCmd()
    }
    return len(this.clients)
}

// Close all ssh clients.
func (this *AgentPool) Close() {
    for _, sshClient := range this.clients {
        sshClient.Close()
    }
}

// SSH connection session instance
type SSHClient struct {
    user       string
    host       string
    running    bool
    passwd     string

    outputChan chan<- *CmdOutput

    sshConfig  *ssh.ClientConfig
    client     *ssh.Client
}

func NewSSHClient(authType, user, passwd, host string, o chan<- *CmdOutput) (*SSHClient, error) {
    var authMethod ssh.AuthMethod

    if authType == "password" {
        if passwd == "" {
            return nil, fmt.Errorf("Host: %s, Failed to connect, password empty", host)
        }
        authMethod = ssh.Password(passwd)
    } else if authType == "publickey" {
        if method, err := publicKeyFile(); err == nil {
            authMethod  = method
        } else {
            return nil, fmt.Errorf("Host: %s, Failed to connect, %s", host, err.Error())
        }
    } else {
        authMethod      = ssh.Password(passwd)
    }

    // Get ssh auth config
    sshConfig := &ssh.ClientConfig{
        User  : user,
        Auth  : []ssh.AuthMethod{authMethod},
    }

    // Create a new ssh connection
    client, err := sshConnect(host, sshConfig)
    if err != nil {
        return nil, err
    }
    return &SSHClient{
        user       : user,
        host       : host,
        running    : false,
        passwd     : passwd,
        outputChan : o,
        sshConfig  : sshConfig,
        client     : client,
    }, nil
}

// Stop running cmd.
func (this *SSHClient) StopCmd() {
    this.running = false
}

func (this *SSHClient) RunCmd(cmdType, cmd string, t int) {
    if cmdType == "sudo" {
        cmd     = strings.Replace(cmd, "sudo", fmt.Sprintf("echo '%s'| sudo -S", this.passwd), 1)
    }
    this.exec(cmd, t)
}

// Scp command, using sftp protocol.
func (this *SSHClient) RunSftp(dest string, sftpFiles []*SftpFile) {
    if len(sftpFiles) == 0 {
        this.runOutput(nil, fmt.Errorf("No sftp file exists"))
        return
    }
    client, err := sftp.NewClient(this.client)
    if err != nil {
        this.runOutput(nil, err)
        return
    }
    defer client.Close()

    // Make sure the destination filepath exist.
    w := client.Walk(dest)
    for w.Step() {
        if w.Err() != nil {         // dest path not exist.
            this.runOutput(nil, fmt.Errorf("Dest path: %s, no such file or directory", w.Path()))
            return
        }
    }
    this.running = true
    out         := []string{}
    for this.running == true {
        for _, sf := range sftpFiles {
            f, err := client.Create(fmt.Sprintf("%s/%s", sf.Destination, sf.Filename))
            if err != nil {
                out = append(out, fmt.Sprintf("%s, %s", sf.Filename, err.Error()))
                continue
            }
            if _, err := f.Write(sf.File); err != nil {
                out    = append(out, fmt.Sprintf("%s, %s", sf.Filename, err.Error()))
                continue
            }
            if _, err := client.Lstat(sf.Filename); err != nil {
                out    = append(out, fmt.Sprintf("%s, %s", sf.Filename, err.Error()))
                continue
            }

            out = append(out, fmt.Sprintf("%s, %s => %s", sf.Filename, this.host, sf.Destination))
        }
        this.running   = false
    }
    this.runOutput(out, nil)
}


// Exec command.
func (this *SSHClient) exec(cmd string, t int) {
    session, err := newSSHSession(this.client)
    if err != nil {
        this.runOutput(nil, err)
        return
    }
    defer session.Close()

    this.running   = true
    // Redirect stdout & stderr, stdin
    var stdout, stdin bytes.Buffer
    session.Stdout = &stdout
    session.Stderr = &stdout
    session.Stdin  = &stdin
    if err := session.Start(cmd); err != nil {      // Cmd start run
        this.runOutput(nil, err)
        return
    }

    // Listen the cmd stop signal
    stopChan := make(chan error)
    go func() {
        stopChan<- session.Wait()
    }()

    if t > 0 {
        timer := time.NewTimer(time.Duration(t) * time.Second)    // exec timeout
        for this.running == true {
            select {
            case <-timer.C:                      // exec time up, send stop singal & stop cmd
                session.Signal(ssh.SIGINT)
                this.running = false
            case e := <-stopChan:                // received the stop info, exit normal
                err          = e
                this.running = false
            default:
                time.Sleep(10 * time.Microsecond)
            }
        }
    } else {
        for this.running == true {
            select {
            case e := <-stopChan:               // received the stop info, exit normal
                err          = e
                this.running = false
            default:
                time.Sleep(10 * time.Microsecond)
            }
        }
    }
    this.runOutput(this.outputFilter(stdout.Bytes()), err)
}

// Running results given to outputChan.
func (this *SSHClient) runOutput(o []string, err error) {
    op := &CmdOutput{
        Host   : this.host,
        Output : o,
        Error  : err,
    }
    this.outputChan<- op
}

// Filter the output.
func (this *SSHClient) outputFilter(o []byte) []string {
    return []string{string(o)}
}

// Close session & conn.
func (this *SSHClient) Close() {
    this.client.Close()
}

// Connect to host using tcp.
func sshConnect(host string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
    client, err := ssh.Dial("tcp", host, sshConfig)            // Connect to host
    if err != nil {
        return nil, err
    }
    return client, nil
}

// Make a new session.
func newSSHSession(client *ssh.Client) (*ssh.Session, error) {
    // Open new session that acts as an entry point to the remote terminal
    session, err := client.NewSession()
    if err != nil {
        return nil, fmt.Errorf("Failed to create session, %s", err.Error())
    }

    // Before we will be able to run the command on the remote machine, we should create a pseudo
    // terminal on the remote machine. A pseudoterminal (or "pty") is a pair of virtual character
    // devices that provide a bidirectional communication channel.
    modes := ssh.TerminalModes{
        ssh.ECHO          : 0,
        ssh.ISIG          : 1,
        ssh.TTY_OP_ISPEED : 14400,
        ssh.TTY_OP_OSPEED : 14400,
    }

    if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
        session.Close()
        return nil, fmt.Errorf("Request pseudo terminal failed, %s", err.Error())
    }
    return session, nil
}

// Parse public keys.
func publicKeyFile() (ssh.AuthMethod, error) {
    buffer, err := ioutil.ReadFile(PublicKeyPath)
    if err != nil {
        return nil, err
    }

    if key, err := ssh.ParsePrivateKey(buffer); err != nil {
        return nil, err
    } else {
        return ssh.PublicKeys(key), nil
    }
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
