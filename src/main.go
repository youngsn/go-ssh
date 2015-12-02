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

    signal              := NewSignal()
    go signal.Start()           // listening Ctrl+c cmd

    if Password == "" {
        fmt.Printf("%s's password: ", Username)
        Password         = GetPasswd(false)
    }

    SSHAgents            = NewAgentPool(Username, Password, Hosts, OutputChan)
    if len(SSHAgents.Failed) != 0 {
        for _, msg := range SSHAgents.Failed {
            fmt.Println(msg)
        }

        fmt.Println()
    }

    if SSHAgents.Active() == false {
        ErrExit(fmt.Errorf("Can not connect to all clients"))
    }

    oLen                := SSHAgents.Exec(Cmd)
    outputs             := SSHOput.GetOutput(oLen)
    stdout(outputs)

    os.Exit(0)
}


func stdout(outputs []*CmdOutput) {
    if len(outputs) == 0 {
        fmt.Println("no any outputs")
        return
    }

    for _, res := range outputs {
        fmt.Printf("%s:\n", res.Host)

        if res.Error != nil {
            fmt.Println("Error:", res.Error.Error())
        }

        if len(res.Output) == 0 {
            fmt.Println("no outputs")
            continue
        }

        for _, op := range res.Output {
            fmt.Println(op)
        }

        fmt.Println("")
    }

}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
