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
    if err := Initialize(); err != nil {
        ErrExit(err)
    }

    runtime.GOMAXPROCS(1)

    signal         := NewSignal()
    go signal.Run()           // listening Ctrl+c cmd

    if PublicKeyPath == "" && Password == "" {
        fmt.Printf("password for %s: ", Username)
        Password        = GetPasswd(false)
    } else if PublicKeyPath != "" && Password == "" {       // If sudo cmd, needs password
        if IsSudo(Cmd) {
            fmt.Printf("password for %s: ", Username)
            Password    = GetPasswd(false)
        }
    }

    SSHAgents       = NewAgentPool(Username, Password, Hosts, OutputChan)
    if len(SSHAgents.Failed) != 0 {
        for _, msg := range SSHAgents.Failed {
            fmt.Println(msg)
        }

        fmt.Println()
    }

    if SSHAgents.Active() == false {
        ErrExit(fmt.Errorf("Can not connect to all clients"))
    }

    fmt.Println("Start......")
    oLen           := SSHAgents.Exec(Cmd, Timeout)
    outputs        := SSHOput.GetOutput(oLen)
    StdOutput(outputs)
    SSHAgents.Close()

    os.Exit(0)
}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
