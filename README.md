# Elfgate

Batch exec command on servers &amp; written by golang.

## Features

- You can exec command on batch servers.

## Getting started

Get the package and build it.

``` shell
$ go get github.com/youngsn/go-ssh
$ go build -o ssh-logins $GOPATH/github.com/youngsn/go-ssh/src/main.go
```

``` shell
$ ./ssh-logins -c $CONF -d "$CMD"
```

## Config sytax
```toml
Username = ""       # server username
Password = ""       # login password, if don't config, you will enter through stdin
PublicKey = ""      # ssh public authorized key path, if using this, add here

Hosts = [
    "127.0.0.1"         # default port 22
    "127.0.0.2:25"      # port 25
]
```


## Third packages

uses packages.

- [toml config parser](https://github.com/BurntSushi/toml) master
- [golang ssh](https://golang.org/x/crypto/ssh)

## TODO

Interactive cmds & sudo cmds solution.

## Author

**TangYang**
<youngsn.tang@gmail.com>


## License

Released under the [MIT License](https://github.com/youngsn/go-ssh/blob/master/LICENSE).
