package elfgate


import (
    "os"
    "fmt"
    "runtime"
    "strings"
    "io/ioutil"

    "gopkg.in/yaml.v2"
    "github.com/codegangsta/cli"
)


var (
    Username      string
    Password      string
    PublicKeyPath string

    Cmd           string
    Timeout       int
    Hosts         []string

    SSHAgents     *AgentPool
    SSHOput       *SSHOut

    OutputChan    chan *CmdOutput
)

// Init app details.
func AppInit() *cli.App {
    cmdFlag       := []cli.Flag{
        cli.StringFlag{
            Name  : "config, c",
            Value : "",
            Usage : "elfgate config file path",
        },
        cli.StringFlag{
            Name  : "group, g",
            Value : "default",
            Usage : "hosts group name",
        },
        cli.IntFlag{
            Name  : "timeout, t",
            Value : 0,
            Usage : "cmd execute timeout",
        },
    }

    var err error
    app        := cli.NewApp()
    app.Name    = APP_NAME
    app.Usage   = "execute commands on batch servers"
    app.Version = APP_VERSION
    app.Flags   = cmdFlag
    app.Before  = func(c *cli.Context) error {   // Parse cmds & params
        err    := parseParams(c)
        if err != nil {
            cli.ShowAppHelp(c)
        }
        return err
    }
    app.After   = func(c *cli.Context) error {   // If has exec errors, return
        if err != nil {
            cli.ShowAppHelp(c)
        }
        return err
    }
    app.Action  = func(c *cli.Context) {         // Run apps 
        err     = appRun()
    }

    return app
}

// Main running commands.
func appRun() error {
    runtime.GOMAXPROCS(1)
    go NewSignal().Run()           // listen ^C & kill

    if PublicKeyPath == "" && Password == "" {
        Password      = getPasswd()
    } else if PublicKeyPath != "" && Password == "" {       // If sudo cmd, needs password
        if CmdType(Cmd) == "sudo" {
            Password  = getPasswd()
        }
    }

    SSHAgents = NewAgentPool(Username, Password, Hosts, OutputChan)
    if SSHAgents.Active() == false {
        return fmt.Errorf("can not connect any clients")
    }

    if err := SSHAgents.Exec(Cmd, Timeout); err != nil {
        return err
    }
    outputs  := SSHOput.GetOutput(SSHAgents.Len())
    StdOutput(outputs)
    SSHAgents.Close()
    return nil
}

// Parse command line params.
func parseParams(context *cli.Context) error {
    var cfgFile     = context.String("config")
    var group       = context.String("group")
    var timeout     = context.Int("timeout")

    defaultCfgPath := [...]string{"/etc/elfgate.yaml", "./elfgate.yml"}
    if cfgFile == "" {
        for _, cfgPath := range defaultCfgPath {
            if err := FileExist(cfgPath); err == nil {
                cfgFile = cfgPath
                break
            }
        }
    }

    if cfgFile == "" {
        return fmt.Errorf("not found /etc/elfgate.yaml or ./elfgate.yaml, specify config file")
    }

    config    := ConfigStruct{}
    if c, err := ioutil.ReadFile(cfgFile); err != nil {
        return err
    } else {
        if err = yaml.Unmarshal(c, &config); err != nil {
            return err
        }
    }

    Timeout = timeout
    Cmd     = ""
    if len(context.Args()) > 0 {
        Cmd = strings.TrimSpace(strings.Join(context.Args(), " "))
    }
    if Cmd == "" {
        return fmt.Errorf("execute command can not be empty")
    }

    Username        = config.Username
    Password        = config.Password
    PublicKeyPath   = config.PublicKey
    if _, ok := config.Groups[group]; !ok {     // Support multi groups
        return fmt.Errorf("-g, group: %s, not exist", group)
    }

    for name, s := range config.Groups {        // Parse & valid hosts
        if name == group {
            var err error
            if Hosts, err = ParseHosts(s); err != nil {
                return err
            }
            break
        }
    }

    if len(Hosts) == 0 {
        return fmt.Errorf("-g, group: %s, have no valid host", group)
    }

    OutputChan = make(chan *CmdOutput, 10240)
    SSHOput    = NewSSHOut(OutputChan)
    return nil
}

// Get password from stdin
func getPasswd() string {
    fmt.Printf("password for %s: ", Username)
    passwd, err := GetPasswd(false)
    if err != nil {
        fmt.Println()
        fmt.Println(err.Error())
        os.Exit(1)
    }
    return passwd
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
