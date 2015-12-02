package elfgate


// System Signal capture & handle.
// Also Signal will run in main thread util program stopped.
// @AUTHOR tangyang

import (
    "os"
    "time"
    "os/signal"
    "syscall"
)


type Signal struct {
    signalChan       chan os.Signal
}


func NewSignal() *Signal {
    signalChan     := make(chan os.Signal)
    signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)  // 监听interrupt & kill

    return &Signal{
        signalChan : signalChan,
    }
}


func (this *Signal) Start() {
    for {
        signal    := <-this.signalChan
        if signal == syscall.SIGINT || signal == syscall.SIGTERM {
            os.Exit(0)
            return
        }

        time.Sleep(100 * time.Microsecond)
    }
}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
