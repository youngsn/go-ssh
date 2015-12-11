package elfgate


type SftpFile struct {
    Filename        string
    Destination     string
    File            []byte
}


type CmdOutput struct {
    Host            string
    Output          []string
    Error           error
}


type ConfigStruct struct {
    Username        string
    Password        string
    PublicKey       string
    Groups          map[string]t
}

type t struct {
    Hosts           []string
}

/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
