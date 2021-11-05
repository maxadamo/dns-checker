# dns-checker

## Table of contents

1. [Preamble](#preamble)
2. [Compiling the program](#compiling-the-program)
3. [Keepalived and LVS](#keepalived-and-LVS)
4. [Available options](#available-options)
5. [Setting up systemd](#setting-up-systemd)
6. [ToDo](#todo)

## Preamble

This application checks the local DNS and optionally consul and serves the status through a Web page.

It should be run as daemon and it can to be used in conjunction with your UDP load-balancer or you can also use this application from Nagios, Sensu and issue a simple HTTP check.

## Compiling the program

You can install GO and copy/paste the followings:

```shell
git checkout main
git pull
LATEST_TAG=$(git describe --tags $(git rev-list --tags --max-count=1))
PROG_VERSION=${LATEST_TAG:1}
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')
git checkout $LATEST_TAG

go get -ldflags "-s -w -X main.appVersion=${PROG_VERSION} -X main.buildTime=${BUILD_TIME}" .
```

## Keepalived and LVS

For instance, with Keepalived + LVS I am using a configuration as follows:

```txt
HTTP_GET {
  connect_port 10053
  connect_timeout 3
  delay_before_retry 1
  http_protocol 1.1
  nb_get_retry 2
  url {
    digest 6d3bcaba1fff8c5a461669b409c1a6d2
    path /ipv4
  }
}
```

the digest is calculated using this command (`genhash` belongs to keepalived package):

```bash
genhash -s 127.0.0.1 -p 10053 -u /ipv4
```

And you'll get the same digest as mine, because the digest is computed against the small HTML snippet embedded in the `main.go`.

You could also use the HTTP status code: `man keepalived.conf` and search for `status_code`.

## Available options

```shell
$ dns-checker --help
DNS Checker:
  - checks DNS and optionally Consul and report the status on a Web page
  
Usage:
  dns-checker [--dns-port=DNSPORT] [--consul-port=CONSULPORT] --dns-record=DNSRECORD [--consul-record=CONSULRECORD] [--consul] [--verbose] [--listen-port=LISTENPORT]
  dns-checker -h | --help
  dns-checker -b | --build
  dns-checker -v | --version
  
Options:
  -h --help                         Show this screen
  -v --version                      Print version information and exit
  -b --build                        Print version and build information and exit
  --dns-port=DNSPORT                DNS port [default: 53]
  --consul-port=CONSULPORT          Consul port [default: 8600]
  --dns-record=DNSRECORD            DNS record to check. A local record is recommended.
  --consul-record=CONSULRECORD      Consul record to check [default: consul.service.consul]
  --consul                          Check consul DNS as well
  --listen-port=LISTENPORT          Web server port [default: 10053]
  --verbose                         Log also successful connections
```

## Setting up systemd

In this case I am also checking for Consul, and I check the existance of one local record called `dumb-record.dumb.zone` in the DNS and one record called `consul.service.domain.org` in Consul.

It is not sensible to check for a record on a forwarded zone, because there can be a problem in the network, or in he SOA of the other domain and we don't want to bring our DNS down if something else is broken.

```systemd
#
# Start DNS checker web service on port 10053
#
[Unit]
Description=DNS and Consul Checker written in Go
Wants=basic.target
After=basic.target network.target

[Service]
User=root
Group=root
ExecStart=/usr/bin/dns-checker --consul --consul-record=consul.service.domain.org --dns-record=dumb-record.dumb.zone
Restart=on-failure
RestartSec=10
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=dns-checker

[Install]
WantedBy=multi-user.target
```

you don't need to run it as root :-)

## ToDo

Add an option to select the interfaces, IPs.
Now it listens on every interface and every protocol (I have dual stack IPv4 and IPv6) and you have to use a firewall if necessary.
