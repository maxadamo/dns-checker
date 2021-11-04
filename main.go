package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/docopt/docopt-go"
)

type checkStatus struct {
	dnsStatus string
}

const statusHTML string = `<!DOCTYPE html>
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
    <title></title>
  </head>
  <body>
    <p>{{.dnsStatus}}</p>
  </body>
</html>
`

var statusTemplate = template.Must(template.ParseFiles("dns_status.html"))

var (
	buf          bytes.Buffer
	appVersion   string
	buildTime    string
	consulBool   bool
	dnsPort      string
	consulPort   string
	dnsRecord    string
	consulRecord string
)

// check DNS
func checkDNS(proto string, DNSport string, consulPort string, recordDNS string, recordConsul string, isConsul bool) string {

	serverDNSandPort := fmt.Sprintf("127.0.0.1:%v", DNSport)
	serverConsulandPort := fmt.Sprintf("127.0.0.1:%v", consulPort)
	if proto == "ipv6" {
		serverDNSandPort = fmt.Sprintf("[::1]:%v", DNSport)
		serverConsulandPort = fmt.Sprintf("127.0.0.1:%v", consulPort)
	}
	t := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			d := net.Dialer{
				Timeout: time.Millisecond * time.Duration(10000),
			}
			return d.DialContext(ctx, network, serverDNSandPort)
		},
	}
	_, errDNS := t.LookupHost(context.Background(), recordDNS)
	if errDNS != nil {
		return "DNS is DOWN"
	}

	if isConsul {
		r := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(10000),
				}
				return d.DialContext(ctx, network, serverDNSandPort)
			},
		}
		_, errConsulForward := r.LookupHost(context.Background(), recordConsul)
		if errConsulForward != nil {
			return "Forwarding to Consul is NOT working"
		}
		s := &net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * time.Duration(10000),
				}
				return d.DialContext(ctx, network, serverConsulandPort)
			},
		}
		_, errConsulStatus := s.LookupHost(context.Background(), recordConsul)
		if errConsulStatus != nil {
			return "Consul is DOWN"
		}
	}
	return "DNS is UP"
}

func ipv4(w http.ResponseWriter, req *http.Request) {
	ipv4_dns_status := checkDNS("ipv4", dnsPort, consulPort, dnsRecord, consulRecord, consulBool)
	parse := checkStatus{dnsStatus: ipv4_dns_status}
	// tmplt, _ := template.ParseFiles("./dns_status.html")
	fmt.Printf("%v\n", ipv4_dns_status)
	if ipv4_dns_status == "DNS is UP" {
		w.WriteHeader(http.StatusOK)
		statusTemplate.Execute(w, parse)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		statusTemplate.Execute(w, parse)
	}
}

func ipv6(w http.ResponseWriter, req *http.Request) {
	ipv6_dns_status := checkDNS("ipv6", dnsPort, consulPort, dnsRecord, consulRecord, consulBool)
	tmpl := template.New("DNS template")
	tmpl, _ = tmpl.Parse(statusHTML)
	parse := checkStatus{dnsStatus: ipv6_dns_status}
	_ = tmpl.Execute(&buf, parse)

	output := buf.String()
	if ipv6_dns_status == "DNS is UP" {
		fmt.Fprintf(w, "%v\n", output)
	} else {
		http.Error(w, output, http.StatusServiceUnavailable)
	}
}

func main() {

	progName := filepath.Base(os.Args[0])

	usage := fmt.Sprintf(`DNS Checker:
  - checks DNS and optionally Consul and report the status on a Web page
  
Usage:
  %v [--dns-port=DNSPORT] [--consul-port=CONSULPORT] [--dns-record=DNSRECORD] [--consul-record=CONSULRECORD] [--consul] [--ipv6] [--listen-port=LISTENPORT]
  %v -h | --help
  %v -v | --version
  %v -b | --build
  
Options:
  -h --help                         Show this screen
  -v --version                      Print version information and exit
  -b --build                        Print version and build information and exit
  --dns-port=DNSPORT                DNS port [default: 53]
  --consul-port=CONSULPORT          Consul port [default: 8600]
  --dns-record=DNSRECORD            DNS record to check [default: www.geant.org]
  --consul-record=CONSULRECORD      Consul record to check [default: consul.service.consul]
  --consul                          Check consul DNS as well
  --ipv6                            Check IPv6 too
  --listen-port=LISTENPORT          Web server port [default: 10053]
`, progName, progName, progName, progName)

	arguments, _ := docopt.ParseArgs(usage, nil, appVersion)

	if arguments["--build"] == true {
		fmt.Printf("%v version: %v, built on: %v\n", progName, appVersion, buildTime)
		os.Exit(0)
	}

	if arguments["--consul"] == true {
		consulBool = true
	} else {
		consulBool = false
	}

	listenPort := arguments["--listen-port"].(string)

	dnsPort = arguments["--dns-port"].(string)
	consulPort = arguments["--consul-port"].(string)
	dnsRecord = arguments["--dns-record"].(string)
	consulRecord = arguments["--consul-record"].(string)

	http.HandleFunc("/ipv4", ipv4)
	http.HandleFunc("/ipv6", ipv6)

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", listenPort), nil))

}
