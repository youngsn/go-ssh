package elfgate


import(
    "fmt"
    "time"
    "bytes"
    "strings"
    "io/ioutil"

    "golang.org/x/crypto/ssh"
)


type AgentPool struct {
    clients []*SSHClient
    Failed  []string
}


func NewAgentPool(user, passwd string, hosts []string, o chan *CmdOutput) *AgentPool {
    var clients  []*SSHClient
    var failed   []string
    var authType string

    if passwd == "" {
        authType = "publickey"
    } else {
        authType = "password"
    }

    for _, host := range hosts {
        sshClient, err := NewSSHClient(authType, user, passwd, host, o)
        if err != nil {
            failed      = append(failed, err.Error())
        } else {
            clients     = append(clients, sshClient)
        }
    }

    return &AgentPool{
        clients : clients,
        Failed  : failed,
    }
}


// Is there have alive clients
func (this *AgentPool) Active() bool {
    if len(this.clients) == 0 {
        return false
    } else {
        return true
    }
}


// Exec cmds
func (this *AgentPool) Exec(cmd string, t int) int {
    for _, sshClient := range this.clients {
        go sshClient.Exec(cmd, t)
    }

    return len(this.clients)
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
    host            string
    running         bool
    passwd          string

    outputChan      chan<- *CmdOutput

    sshConfig       *ssh.ClientConfig
    client          *ssh.Client
    sshSession      *ssh.Session
}


func NewSSHClient(authType, user, passwd, host string, o chan<- *CmdOutput) (*SSHClient, error) {
    var authMethod ssh.AuthMethod

    if authType == "password" {
        if passwd == "" {
            return nil, fmt.Errorf("Host: %s, Failed to connect, password empty", host)
        }
        authMethod      = ssh.Password(passwd)
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
    sshConfig          := &ssh.ClientConfig{
        User : user,
        Auth : []ssh.AuthMethod{authMethod},
    }

    // Create a new ssh connection
    client, err        := sshConnect(host, sshConfig)
    if err != nil {
        return nil, err
    }
    // Open new session
    session, err       := newSSHSession(host, client)
    if err != nil {
        return nil, err
    }

    return &SSHClient{
        host       : host,
        running    : false,
        passwd     : passwd,
        outputChan : o,
        sshConfig  : sshConfig,
        client     : client,
        sshSession : session,
    }, nil
}


// Stop running cmd.
func (this *SSHClient) StopCmd() error {
    this.running        = false
    return this.sshSession.Signal(ssh.SIGINT)   // send ctrl+c cmd
}


// Support sudo commands nointeractive.
func (this *SSHClient) commandHandle(cmd string) string {
    fcmd    := cmd
    if IsSudo(cmd) {
       fcmd  = strings.Replace(cmd, "sudo", fmt.Sprintf("echo '%s' | sudo -S", this.passwd), 1)
    }

    return fcmd
}


// Exec command.
func (this *SSHClient) Exec(cmd string, t int) {
    cmd                    = this.commandHandle(cmd)
    this.running           = true

    // Redirect stdout & stderr, stdin
    var stdout, stdin bytes.Buffer
    this.sshSession.Stdout = &stdout
    this.sshSession.Stderr = &stdout
    this.sshSession.Stdin  = &stdin

    if err := this.sshSession.Start(cmd); err != nil {      // Cmd start run
        op := &CmdOutput{
            Host   : this.host,
            Output : []string{},
            Error  : err,
        }
        this.outputChan<- op

        return
    }

    // Listen the cmd stop signal
    stopChan    := make(chan error)
    go func() {
        stopChan<- this.sshSession.Wait()
    }()

    var err error
    if t > 0 {
        timer            := time.NewTimer(time.Duration(t) * time.Second)    // exec timeout
        for this.running == true {
            select {
            case <-timer.C:                      // exec time up, send stop singal & stop cmd
                this.sshSession.Signal(ssh.SIGINT)
                this.running   = false
            case e := <-stopChan:                // received the stop info, exit normal
                err            = e
                this.running   = false
            default:
                time.Sleep(10 * time.Microsecond)
            }
        }
    } else {
        for this.running == true {
            select {
            case e := <-stopChan:               // received the stop info, exit normal
                err            = e
                this.running   = false
            default:
                time.Sleep(10 * time.Microsecond)
            }
        }
    }

    op  := &CmdOutput{
        Host   : this.host,
        Output : this.outputFilter(stdout.Bytes()),
        Error  : err,
    }

    this.outputChan<- op
}


// Filter the output.
func (this *SSHClient) outputFilter(o []byte) []string {
    // oline    := []byte{}
    // outputs  := []string{}
    // for _, c := range stdout.Bytes() {
    //     if string(c) == "\n" {          // \n char set
    //         outputs   = append(outputs, string(oline))
    //         oline     = []byte{}
    //     } else {
    //         oline     = append(oline, c)
    //     }
    // }
    return []string{string(o)}
}


// Reconnect and create session.
func (this *SSHClient) newSession() error {
    var err error
    this.sshSession, err = newSSHSession(this.host, this.client)
    if err != nil {
        return err
    } else {
        return nil
    }
}


// Close session & conn.
func (this *SSHClient) Close() {
    this.sshSession.Close()
    this.client.Close()
}


// Connect to host using tcp.
func sshConnect(host string, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
    client, err          := ssh.Dial("tcp", host, sshConfig)            // Connect to host
    if err != nil {
        return nil, fmt.Errorf("Host: %s, Failed to connect, %s", host, err.Error())
    }

    return client, nil
}


// Make a new session.
func newSSHSession(host string, client *ssh.Client) (*ssh.Session, error) {
    // Open new session that acts as an entry point to the remote terminal
    session, err       := client.NewSession()
    if err != nil {
        return nil, fmt.Errorf("Host: %s, Failed to create session, %s", host, err.Error())
    }

    // Before we will be able to run the command on the remote machine, we should create a pseudo
    // terminal on the remote machine. A pseudoterminal (or "pty") is a pair of virtual character
    // devices that provide a bidirectional communication channel.
    modes              := ssh.TerminalModes{
        ssh.ECHO          : 0,
        ssh.ISIG          : 1,
        ssh.TTY_OP_ISPEED : 14400,
        ssh.TTY_OP_OSPEED : 14400,
    }

    if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
        session.Close()
        return nil, fmt.Errorf("Host: %s, Request pseudo terminal failed, %s", host, err.Error())
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
