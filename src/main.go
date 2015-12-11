package main


// Program start here.
// @AUTHOR tangyang

import (
    "os"
    "fmt"
    "runtime"

    . "elfgate"
)


func main() {
    var err error
    if err = Initialize(); err != nil {
        ErrExit(err)
    }

    runtime.GOMAXPROCS(1)

    signal           := NewSignal()
    go signal.Run()           // listen ^C & kill

    if PublicKeyPath == "" && Password == "" {
        Password      = getPasswd()
    } else if PublicKeyPath != "" && Password == "" {       // If sudo cmd, needs password
        if CmdType(Cmd) == "sudo" {
            Password  = getPasswd()
        }
    }

    SSHAgents         = NewAgentPool(Username, Password, Hosts, OutputChan)
    if SSHAgents.Active() == false {
        ErrExit(fmt.Errorf("Can not connect to all clients"))
    }

    if err := SSHAgents.Exec(Cmd, Timeout); err != nil {
        ErrExit(err)
    }
    outputs        := SSHOput.GetOutput(SSHAgents.Len())
    StdOutput(outputs)
    SSHAgents.Close()

    os.Exit(0)
}


func getPasswd() string {
    fmt.Printf("password for %s: ", Username)
    passwd, err := GetPasswd(false)
    if err != nil {
        fmt.Println()
        fmt.Println(err.Error())
        os.Exit(0)
    }

    return passwd
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
