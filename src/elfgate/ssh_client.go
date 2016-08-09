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
func (t *AgentPool) Active() bool {
    if len(t.clients) == 0 {
        return false
    } else {
        return true
    }
}

// Return Alive client len.
func (t *AgentPool) Len() int {
    return len(t.clients)
}

// Exec cmds
func (t *AgentPool) Exec(cmd string, timeout int) error {
    cmdType    := CmdType(cmd)
    if cmdType == "scp" {
        fs, err := ScpCmdProc(t.user, Cmd)
        if err != nil {
            return err
        }

        for _, sshClient := range t.clients {
            go sshClient.RunScp(fs)
        }
    } else {
        for _, sshClient := range t.clients {
            go sshClient.RunCmd(cmdType, cmd, timeout)
        }
    }
    return nil
}

// Stop running cmds
func (t *AgentPool) StopCmds() int {
    for _, sshClient := range t.clients {
        go sshClient.StopCmd()
    }
    return len(t.clients)
}

// Close all ssh clients.
func (t *AgentPool) Close() {
    for _, sshClient := range t.clients {
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
    conn       *ssh.Client
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
        authMethod = ssh.Password(passwd)
    }

    // Get ssh auth config
    sshConfig := &ssh.ClientConfig{
        User  : user,
        Auth  : []ssh.AuthMethod{authMethod},
    }

    // Create a new ssh connection
    conn, err := sshConnect(host, sshConfig)
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
        conn       : conn,
    }, nil
}

// Stop running cmd.
func (t *SSHClient) StopCmd() {
    t.running = false
}

func (t *SSHClient) RunCmd(cmdType, cmd string, timeout int) {
    if cmdType == "sudo" {
        cmd    = strings.Replace(cmd, "sudo", fmt.Sprintf("echo '%s'| sudo -S", t.passwd), 1)
    }
    t.exec(cmd, timeout)
}

// Scp command, using sftp protocol.
func (t *SSHClient) RunScp(cpFiles []*CpFile) {
    if len(cpFiles) == 0 {
        t.runOutput(nil, fmt.Errorf("cp file not exists"))
        return
    }
    client, err := sftp.NewClient(t.conn)
    if err != nil {
        t.runOutput(nil, err)
        return
    }
    defer client.Close()

    // Start cp files to dest
    t.running = true
    out      := []string{}
    for t.running == true {
        for _, cf := range cpFiles {        // cp all files to destination
            output, _ := t.scp2Dest(client, cf)
            out        = append(out, output)
        }
        t.running      = false
    }
    t.runOutput(out, nil)
}

// Scp one cpFile to destination.
// If destination is not exist, be regarded as a dir default, then make it.
// If destination is a dir, make file in it.
// If destination is a file, cover it.
func (t *SSHClient) scp2Dest(client *sftp.Client, cpFile *CpFile) (string, bool) {
    filename := cpFile.Filename
    dest     := cpFile.Destination
    content  := cpFile.Content

    // Dest dir not exist, make it.
    destType   := "file"
    if fi, err := client.Lstat(dest); err != nil {      // Dest not exist，make as a dir.
        if err := client.Mkdir(dest); err != nil {
            return fmt.Sprintf("mkdir %s, %s", dest, err.Error()), false
        }
    } else {
        if fi.IsDir() == true {
            destType   = "dir"
        }
    }
    destF       := dest
    if destType == "dir" {          // dir, build filepath
        destF    = fmt.Sprintf("%s/%s", dest, filename)
    }

    f, err := client.Create(destF)
    if err != nil {
        return fmt.Sprintf("create: %s, %s", destF, err.Error()), false
    }
    if _, err := f.Write(content); err != nil {
        return fmt.Sprintf("write: %s, %s", destF, err.Error()), false
    }
    return fmt.Sprintf("cp: %s, %s => %s", filename, t.host, dest), true
}

// Exec user command in timeout seconds.
func (t *SSHClient) exec(cmd string, timeout int) {
    session, err := newSSHSession(t.conn)
    if err != nil {
        t.runOutput(nil, err)
        return
    }
    defer session.Close()
    t.running = true

    // Redirect stdout & stderr, stdin
    var stdout, stdin bytes.Buffer
    session.Stdout = &stdout
    session.Stderr = &stdout
    session.Stdin  = &stdin
    if err := session.Start(cmd); err != nil {      // Cmd start run
        t.runOutput(nil, err)
        return
    }

    // Listen the cmd stop signal
    stopChan := make(chan error)
    go func() {
        stopChan<- session.Wait()
    }()

    if timeout > 0 {
        timer := time.NewTimer(time.Duration(timeout) * time.Second)    // exec timeout
        for t.running == true {
            select {
            case <-timer.C:                      // exec time up, send stop singal & stop cmd
                session.Signal(ssh.SIGINT)
                t.running = false
            case e := <-stopChan:                // received the stop info, exit normal
                err       = e
                t.running = false
            default:
                time.Sleep(10 * time.Microsecond)
            }
        }
    } else {
        for t.running == true {
            select {
            case e := <-stopChan:               // received the stop info, exit normal
                err       = e
                t.running = false
            default:
                time.Sleep(10 * time.Microsecond)
            }
        }
    }
    t.runOutput(t.outputFilter(stdout.Bytes()), err)
}

// Running results given to outputChan.
func (t *SSHClient) runOutput(o []string, err error) {
    op := &CmdOutput{
        Host   : t.host,
        Output : o,
        Error  : err,
    }
    t.outputChan<- op
}

// Filter the output.
func (t *SSHClient) outputFilter(o []byte) []string {
    return []string{string(o)}
}

// Close session & conn.
func (t *SSHClient) Close() {
    t.conn.Close()
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

    if err := session.RequestPty("xterm", 1366, 768, modes); err != nil {
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
