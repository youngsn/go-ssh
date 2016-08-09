package elfgate


// All tool funcs belongs here.
// @author tangyang
import (
    "os"
    "io/ioutil"
    "fmt"
    "regexp"
    "strconv"
    "strings"
    "syscall"

    "golang.org/x/crypto/ssh/terminal"
)

// Parse Hosts & add default ssh ports.
// Supported simple preg, just like: 192.168.1.[1-100]:22.
func ParseHosts(hosts []string) ([]string, error) {
    parsedHosts  := []string{}
    for _, host  := range hosts {
        if ok, _ := regexp.MatchString(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(\[\d+\-\d+\])(\:(\d+))?$`, host); ok {  // x.x.x.[x-x](:x)
            reg, _ := regexp.Compile(`^(.+\.)\[(\d+)\-(\d+)\](\:\d+)?$`)
            ps     := reg.FindStringSubmatch(host)
            s, _   := strconv.Atoi(ps[2])
            e, _   := strconv.Atoi(ps[3])
            if s > e || s < 0 || e < 0 || s > 255 || e > 255 {
                return nil, fmt.Errorf("%s, [%d-%d] not valid", host, s, e)
            }
            for i := s; i <= e; i ++ {
                h := ""
                if ps[4] == "" {
                    h = fmt.Sprintf("%s%d:22", ps[1], i)
                } else {
                    h = fmt.Sprintf("%s%d%s", ps[1], i, ps[4])
                }
                parsedHosts = append(parsedHosts, h)
            }
        } else if ok, _ := regexp.MatchString(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$`, host); ok {  // x.x.x.x
            parsedHosts  = append(parsedHosts, fmt.Sprintf("%s:22", host))
        } else if ok, _ := regexp.MatchString(`^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)(\:(\d+))?$`, host); ok {  // x.x.x.x(:x)
            parsedHosts  = append(parsedHosts, host)
        } else {
            return nil, fmt.Errorf("%s, not valid IP", host)
        }
    }

    filter   := []string{}
    hs       := map[string]string{}
    for _, h := range parsedHosts {
        if _, ok := hs[h]; !ok {         // delete from parsedHosts
            hs[h]  = h
            filter = append(filter, h)
        }
    }
    return filter, nil
}

// Output results to stdout.
func StdOutput(outputs []*CmdOutput) {
    if len(outputs) == 0 {
        fmt.Println("no outputs")
        return
    }
    for _, o := range outputs {
        if o.Error != nil {           // Error outputs.
            fmt.Printf("%s | failed >>\n", o.Host)
            fmt.Printf("return: %s\n", o.Error.Error())
            continue
        }

        // Success outputs.
        fmt.Printf("%s | success >>\n", o.Host)
        if o.Output == nil || len(o.Output) == 0 {
            fmt.Println()
            continue
        }
        for _, op := range o.Output {
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

// Judge the command sudo or not.
func CmdType(cmd string) string {
    if m, _ := regexp.MatchString("^sudo .+$", cmd); m {
        return "sudo"
    } else if m, _ := regexp.MatchString("^scp .+$", cmd); m {
        return "scp"
    } else {
        return "normal"
    }
}

// Proc the scp command.
func ScpCmdProc(user string, cmd string) ([]*CpFile, error) {
    reg, _ := regexp.Compile(`^scp\s+(.+)\s+(.+)$`)
    ps     := reg.FindStringSubmatch(cmd)

    // path handle
    src    := strings.Trim(ps[1], " ")
    dest   := strings.Trim(ps[2], " ")
    if strings.Contains(src, "~") {         // replace ~ to /home/user
        src  = strings.Replace(src, "~", fmt.Sprintf("/home/%s", user), -1)
    }
    if strings.Contains(dest, "~") {
        dest = strings.Replace(dest, "~", fmt.Sprintf("/home/%s", user), -1)
    }

    if src == "" || dest == "" {
        return nil, fmt.Errorf("%s, error command", cmd)
    } else if err := FileExist(src); err != nil {
        return nil, err
    }

    filename     := ""
    ts           := strings.Split(src, "/")
    filename, ts  = ts[len(ts) - 1], ts[:len(ts) - 1]
    fd, err      := os.Open(src)
    if err != nil {
        return nil, err
    }
    contents, err := ioutil.ReadAll(fd)
    if err != nil {
        return nil, err
    }

    cpFile := &CpFile{
        Filename    : filename,
        Destination : dest,
        Content     : contents,
    }
    return []*CpFile{cpFile}, nil
}

// If file or path exist.
func FileExist(path string) error {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        return err
    }
    return nil
}

// If filepath exists, will auto create one if not exist.
func FilePathExist(path string) error {
    if _, err := os.Stat(path); os.IsNotExist(err) {
        if err := os.Mkdir(path, 0775); err != nil {
            return err
        }
    }
    return nil
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
