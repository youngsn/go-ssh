package elfgate


// Program & global param inits here.
import (
    "fmt"
    "flag"
    "strings"
    "io/ioutil"

    "gopkg.in/yaml.v2"
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
    var cfgFile     = flag.String("c", "/etc/elfgate.yaml", "elfgate config file")
    var cmd         = flag.String("d", "", "execute command")
    var group       = flag.String("g", "default", "host group name")
    var timeout     = flag.Int("t", 0, "execute timeout")       // 0 means no timeout
    flag.Parse()

    c, err         := ioutil.ReadFile(*cfgFile)
    if err != nil {
        return err
    }

    config         := ConfigStruct{}
    if err = yaml.Unmarshal(c, &config); err != nil {
        return err
    }

    if strings.TrimSpace(*cmd) == "" {
        return fmt.Errorf("Usage: -d 'exec cmd' -c conf -t timeout -s cluster")
    }

    Cmd             = strings.TrimSpace(*cmd)
    Timeout         = *timeout

    Username        = config.Username
    Password        = config.Password
    PublicKeyPath   = config.PublicKey

    // Support multi clusters
    if _, ok := config.Groups[*group]; !ok {
        return fmt.Errorf("group: %s, not exist", *group)
    }

    // Parse & valid hosts
    for name, s := range config.Groups {
        if name == *group {
            if Hosts, err = ParseHosts(s); err != nil {
                return err
            }

            break
        }
    }

    if len(Hosts) == 0 {
        return fmt.Errorf("Group: %s, no valid hosts", *group)
    }

    OutputChan      = make(chan *CmdOutput, 10240)
    SSHOput         = NewSSHOut(OutputChan)

    return nil
}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
