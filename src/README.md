# Elfgate

Batch exec command on servers &amp; written by golang.

## Features

- Exec command on cluster servers.
- Server clusters support.
- Hosts support simple preg(just IPs).
- sftp supported, can upload file to batch servers(NOTE: supported file only).

## Getting started

Get the package, add the src to workspace path and build it.

But, first you should get golang ssh packages

```shell
$ git clone github.com/youngsn/go-ssh
$ go build -o ssh-logins -v -x $GOPATH/go-ssh/src/main.go
$ cp go-ssh/elfgate.yml /etc/
$ sudo cp ssh-logins /usr/local/bin/
$ ssh-logins "uptime"       # example
```

Usage:

Execute command just like below, the outputs will print on screen or using > redirect to file.

Now also sudo command are supported, but at first you should enter password if not config pasword.

- -c, config file location, default: /etc/elfgate.yml
- -t, command timeout, default: 0, no timeout
- -g, groups that execute commands, default: default
- enter command directly.

```shell
$ ssh-logins -c $CONF -t $TIMEOUT -g $GROUP "$CMD"
```

If you want to upload file to batch server, just do like below. Very easy and now only supported file, not directory.

```shell
$ ssh-logins -c $CONF -g $GROUP "sftp $LOCAL_PATH $REMOTE_PATH"
```

## Config syntax
```yaml
username: tangyang       # server username
password:                # login password, if don't config, you will enter through stdin
public_key:              # ssh public authorized key path, if using this, add here

groups:
    default:             # Group default
        - "127.0.0.[1-5]"        # simple preg support
        - "127.0.0.[6-7]:233"
        - "127.0.0.8"            # default port 22
        - "127.0.0.9:25"         # port 25
    example:             # Group example
        - "127.0.0.2"
        - "127.0.0.3:25"
```


## Third packages

Use below third packages.

- [go-yaml](https://github.com/go-yaml/yaml) v2
- [cli](https://github.com/codegangsta/cli)
- [golang.org/x/crypto/ssh](https://github.com/golang/crypto)
- [github.com/pkg/sftp](https://github.com/pkg/sftp)

## TODO

Interactive cmds solution.

## Author

**TangYang**
<youngsn.tang@gmail.com>

## License

Released under the [MIT License](https://github.com/youngsn/go-ssh/blob/master/LICENSE).
