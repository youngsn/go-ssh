package elfgate


// Program & global param inits here.
import (
    "fmt"
    "flag"
    "strings"

    "github.com/BurntSushi/toml"
)


var (
    PublicKeyPath        string

    Username             string
    Password             string
    Cmd                  string
    Timeout              int
    Hosts                []string

    SSHAgents            *AgentPool
    SSHOput              *SSHOut

    OutputChan           chan *CmdOutput
)


// Config file parse & global vars init(in common & there)
func Initialize() error {
    var config *ConfigStruct
    var err error

    var cfgFile     = flag.String("c", "hosts.toml", "hosts config list")
    var cmd         = flag.String("d", "", "execute command")
    var timeout     = flag.Int("t", 0, "execute timeout")       // 0 means no timeout
    flag.Parse()

    if _, err = toml.DecodeFile(*cfgFile, &config); err != nil {
        return err
    }

    if *cmd == "" {
        return fmt.Errorf("Usage: -d 'exec cmd'; -c hosts; -t timeout")
    }

    Cmd             = *cmd
    Timeout         = *timeout

    Username        = config.Username
    Password        = config.Password
    PublicKeyPath   = config.PublicKey
    Hosts           = []string{}
    for _, v := range config.Hosts {             // add default port
        if !strings.Contains(v, ":") {
            v = v + ":22"
        }
        Hosts       = append(Hosts, v)
    }

    OutputChan      = make(chan *CmdOutput, 10240)
    SSHOput         = NewSSHOut(OutputChan)

    return nil
}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
