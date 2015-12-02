package elfgate


import(
    "fmt"
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


func (this *AgentPool) Exec(cmd string) int {
    for _, sshClient := range this.clients {
        go sshClient.Exec(cmd)
    }

    return len(this.clients)
}


func (this *AgentPool) Close() {
    for _, sshClient := range this.clients {
        sshClient.Close()
    }
}


// SSH connection session instance
type SSHClient struct {
    host            string

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
        outputChan : o,
        sshConfig  : sshConfig,
        client     : client,
        sshSession : session,
    }, nil
}


func (this *SSHClient) Exec(cmd string) {
    if this.sshSession.Stdout != nil || this.sshSession.Stderr != nil {
        this.newSession()
    }

    var op *CmdOutput
    line     := []byte{}
    output   := []string{}

    o, err   := this.sshSession.CombinedOutput(cmd)
    for _, c := range o {
        if string(c) == "\n" {          // \n char set
            output    = append(output, string(line))
            line      = []byte{}
        } else {
            line      = append(line, c)
        }
    }

    op  = &CmdOutput{
        Host   : this.host,
        Output : output,
        Error  : err,
    }

    this.outputChan<- op
}


// Reconnect and create session
func (this *SSHClient) newSession() error {
    var err error
    this.sshSession, err = newSSHSession(this.host, this.client)
    if err != nil {
        return err
    } else {
        return nil
    }
}


// Close session
func (this *SSHClient) Close() {
    this.sshSession.Close()
}


// Connect to host using tcp
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
        ssh.TTY_OP_ISPEED : 14400,
        ssh.TTY_OP_OSPEED : 14400,
    }

    if err := session.RequestPty("xterm", 80, 40, modes); err != nil {
        session.Close()
        return nil, fmt.Errorf("Host: %s, Request pseudo termial failed, %s", host, err.Error())
    }

    return session, nil
}


// Parse public keys
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
