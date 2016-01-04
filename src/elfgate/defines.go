package elfgate


const VERSION       = "1.2.0"


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
    PublicKey       string `yaml:"public_key"`
    Groups          map[string][]string
}


/* vim: set expandtab ts=4 sw=4 sts=4 tw=100: */
