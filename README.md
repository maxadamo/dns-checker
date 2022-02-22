# dns-checker

## Table of contents

1. [Preamble](#preamble)
2. [Compiling the program](#compiling-the-program)
3. [Keepalived and LVS](#keepalived-and-LVS)
4. [Available options](#available-options)
5. [Setting up systemd](#setting-up-systemd)

## Preamble

What problems this application tries to solve? UDP can't be easily checked. This application checks the local DNS and optionally Consul DNS and serves the status through a Web page.

I use it in conjunction with LVS and my understanding is that LVS does not allow to run multiple instances of the same check. For instance, LVS has a `DNS_CHECK` statement, but in my case I need to run it multiple times, to check either the DNS and Consul DNS.

This application runs as a daemon on the same machine where the DNS is running and it can be used in conjunction with your UDP load-balancer to check the status of your DNS.

You can also use it from Nagios, Sensu and issue a simple HTTP check.

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

And if you receive a 200 status code, you'll get the same digest as mine, because the digest is computed against the small HTML snippet embedded in the `main.go`.

You could also use the HTTP status code: `man keepalived.conf` and search for `status_code`.

## Available options

You can check the options as follows:

```shell
$ dns-checker --help
DNS Checker:
  - checks DNS and optionally Consul and report the status on a Web page
  
Usage:
  dns-checker --dns-record=DNSRECORD [--dns-port=DNSPORT] [--consul-port=CONSULPORT] [--consul-record=CONSULRECORD] [--consul] [--verbose] [--listen-port=LISTENPORT] [--listen-address=LISTENADDRESS]
  dns-checker -h | --help
  dns-checker -b | --build
  dns-checker -v | --version
  
Options:
  -h --help                         Show this screen
  -v --version                      Print version information and exit
  -b --build                        Print version and build information and exit
  --dns-record=DNSRECORD            DNS record to check. A local record is recommended.
  --dns-port=DNSPORT                DNS port [default: 53]
  --consul-port=CONSULPORT          Consul port [default: 8600]
  --consul-record=CONSULRECORD      Consul record to check [default: consul.service.consul]
  --consul                          Check consul DNS as well
  --listen-port=LISTENPORT          Web server port [default: 10053]
  --listen-address=LISTENADDRESS    Web server address. Check Go net/http documentation [default: any]
  --verbose                         Log also successful connections
```

Once it is installed you can check the status using curl (with `curl -I` you get the status code):

```bash
curl http://localhost:10053/ipv4
```

## Setting up systemd

In this case I am also checking Consul, and I check the existance of one local record called `dumb-record.dumb.zone` in the DNS and one record called `consul.service.domain.org` in Consul.

It is not sensible to check for a record on a forwarded zone, because there can be a problem elsewhere (in the network, or in he SOA of the other domain) and we don't want to bring our DNS down if something else is broken.

In this case I run it as `unbound` user because I use unbound:

```systemd
#
# Start DNS checker web service on port 10053
#
[Unit]
Description=DNS and Consul Checker written in Go
Wants=basic.target
After=basic.target network.target

[Service]
User=unbound
Group=unbound
ExecStart=/usr/bin/dns-checker --consul --consul-record=consul.service.domain.org --dns-record=dumb-record.dumb.zone
Restart=on-failure
RestartSec=10
StandardOutput=syslog
StandardError=syslog
SyslogIdentifier=dns-checker

[Install]
WantedBy=multi-user.target
```

