package elfgate


// All tool funcs belongs here.
// @author tangyang

import (
    "os"
    "fmt"
    "syscall"

    "golang.org/x/crypto/ssh/terminal"
)


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
