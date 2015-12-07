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
    var cluster     = flag.String("s", "default", "host cluster")
    var timeout     = flag.Int("t", 0, "execute timeout")       // 0 means no timeout
    flag.Parse()

    if _, err = toml.DecodeFile(*cfgFile, &config); err != nil {
        return err
    }

    if *cmd == "" {
        return fmt.Errorf("Usage: -d 'exec cmd'; -c hosts; -t timeout; -s cluster")
    }

    Cmd             = *cmd
    Timeout         = *timeout

    Username        = config.Username
    Password        = config.Password
    PublicKeyPath   = config.PublicKey

    // support multi clusters
    for name, s := range config.Hosts {
        if name == *cluster {
            for _, h := range s.Hosts {
                if !strings.Contains(h, ":") {
                    h    = h + ":22"
                }
                Hosts    = append(Hosts, h)
            }
            break
        }
    }

    OutputChan      = make(chan *CmdOutput, 10240)
    SSHOput         = NewSSHOut(OutputChan)

    return nil
}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
