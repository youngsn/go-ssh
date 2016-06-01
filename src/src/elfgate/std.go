package elfgate


import(
    "os"
    "fmt"
    "time"
    "bufio"
    "strings"
)

// getPasswd returns the input read from terminal.
// If masked is true, typing will be matched by asterisks on the screen.
// Otherwise, typing will echo nothing.
func GetPasswd(masked bool) (string, error) {
    var pass, bs, mask []byte
    if masked {
        bs   = []byte("\b \b")
        mask = []byte("*")
    }

    for {
        if v := Getch(); v == 127 || v == 8 {
            if l := len(pass); l > 0 {
                pass = pass[:(l - 1)]
                os.Stdout.Write(bs)
            }
        } else if v == 13 || v == 10 {
            break
        } else if v == 3 {          // ^C, return exit flag
            return "", fmt.Errorf("exit")
        } else if v != 0 {
            pass     = append(pass, v)
            os.Stdout.Write(mask)
        }
    }
    println()
    return string(pass), nil
}

// Reading from keyboard
type Stdin struct {
    reader *bufio.Reader
}

func NewStdin() *Stdin {
    reader := bufio.NewReader(os.Stdin)
    return &Stdin{
        reader : reader,
    }
}

func (this *Stdin) GetInput() string {
    input, _ := this.reader.ReadString('\n')
    return strings.TrimSpace(input)
}

// Getting response from OutputChan
type SSHOut struct {
    running    bool
    outputChan <-chan *CmdOutput
}

func NewSSHOut(o <-chan *CmdOutput) *SSHOut {
    return &SSHOut{
        running    : false,
        outputChan : o,
    }
}

func (this *SSHOut) Stop() {
    this.running = false
}

func (this *SSHOut) GetOutput(length int) []*CmdOutput {
    outputs      := []*CmdOutput{}
    this.running  = true
    for this.running == true {
        select {
        case op := <-this.outputChan:
            outputs          = append(outputs, op)
            if len(outputs) == length {
                this.running = false
            }
        default:
            time.Sleep(10 * time.Microsecond)
        }
    }
    return outputs
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
