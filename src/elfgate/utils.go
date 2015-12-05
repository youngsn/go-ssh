package elfgate


// All tool funcs belongs here.
// @author tangyang

import (
    "os"
    "fmt"
    "regexp"
    "syscall"

    "golang.org/x/crypto/ssh/terminal"
)


func StdOutput(outputs []*CmdOutput) {
    if len(outputs) == 0 {
        fmt.Println("no any outputs")
        return
    }

    for _, res := range outputs {
        if res.Error != nil {
            fmt.Printf("%s | failed | %s >>\n", res.Host, res.Error.Error())
        } else {
            fmt.Printf("%s | success >>\n", res.Host)
        }

        if len(res.Output) == 0 {
            fmt.Println()
            continue
        }

        for _, op := range res.Output {
            fmt.Println(op)
        }
    }
}


// Get char from terminal
func Getch() byte {
    if oldState, err := terminal.MakeRaw(0); err != nil {
        panic(err.Error())
    } else {
        defer terminal.Restore(0, oldState)
    }

    var buf [1]byte
    if n, err := syscall.Read(0, buf[:]); n == 0 || err != nil {
        panic(err.Error())
    }

    return buf[0]
}


func IsSudo(cmd string) bool {
    if m, _ := regexp.MatchString("^sudo .+$", cmd); m {
        return true
    }

    return false
}

// If filepath exists, will auto create one if not exist.
func FilePathExist(path string) error {
    if _, err  := os.Stat(path); os.IsNotExist(err) {
        if err := os.Mkdir(path, 0775); err != nil {
            return err
        }
    }

    return nil
}


func ErrExit(err error) {
    fmt.Println(err.Error())
    os.Exit(1)
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
