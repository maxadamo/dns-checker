package main

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/docopt/docopt-go"
)

// if the below HTML snippet changes a new MD5 digest code has to be used from LVS
var statusTemplate, _ = template.New("status template").Parse(`<!DOCTYPE html>
<html>
  <head>
    <meta http-equiv="content-type" content="text/html; charset=UTF-8">
    <title></title>
  </head>
  <body>
    <p>{{.}}</p>
  </body>
</html>
`)

var (
	appVersion    string
	buildTime     string
	consulBool    bool
	dnsPort       string
	consulPort    string
	dnsRecord     string
	consulRecord  string
	WarningLogger *log.Logger
	InfoLogger    *log.Logger
	ErrorLogger   *log.Logger
	verbose       bool
)

func init() {
	InfoLogger = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime)
	WarningLogger = log.New(os.Stdout, "WARNING: ", log.Ldate|log.Ltime)
	ErrorLogger = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime)
}

// check DNS
func checkDNS(proto string, DNSport string, consulPort string, recordDNS string, recordConsul string, isConsul bool) string {

	serverDNSandPort := fmt.Sprintf("127.0.0.1:%v", DNSport)
	serverConsulandPort := fmt.Sprintf("127.0.0.1:%v", consulPort)
	if proto == "ipv6" {
		serverDNSandPort = fmt.Sprintf("[::1]:%v", DNSport)
		serverConsulandPort = fmt.Sprintf("[::1]:%v", consulPort)
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

// serve directory ipv4
func ipv4(w http.ResponseWriter, req *http.Request) {
	ipv4_dns_status := checkDNS("ipv4", dnsPort, consulPort, dnsRecord, consulRecord, consulBool)
	if ipv4_dns_status == "DNS is UP" {
		if verbose {
			InfoLogger.Println(ipv4_dns_status)
		}
		w.WriteHeader(http.StatusOK)
		statusTemplate.Execute(w, template.HTML(ipv4_dns_status))
	} else {
		WarningLogger.Println(ipv4_dns_status)
		w.WriteHeader(http.StatusServiceUnavailable)
		statusTemplate.Execute(w, template.HTML(ipv4_dns_status))
	}
}

// serve directory ipv6
func ipv6(w http.ResponseWriter, req *http.Request) {
	ipv6_dns_status := checkDNS("ipv6", dnsPort, consulPort, dnsRecord, consulRecord, consulBool)
	if ipv6_dns_status == "DNS is UP" {
		if verbose {
			InfoLogger.Println(ipv6_dns_status)
		}
		w.WriteHeader(http.StatusOK)
		statusTemplate.Execute(w, template.HTML(ipv6_dns_status))
	} else {
		WarningLogger.Println(ipv6_dns_status)
		w.WriteHeader(http.StatusServiceUnavailable)
		statusTemplate.Execute(w, template.HTML(ipv6_dns_status))
	}
}

func main() {

	progName := filepath.Base(os.Args[0])

	usage := fmt.Sprintf(`DNS Checker:
  - checks DNS and optionally Consul and report the status on a Web page
  
Usage:
  %v [--dns-port=DNSPORT] [--consul-port=CONSULPORT] --dns-record=DNSRECORD [--consul-record=CONSULRECORD] [--consul] [--verbose] [--listen-port=LISTENPORT]
  %v -h | --help
  %v -b | --build
  %v -v | --version
  
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
`, progName, progName, progName, progName)

	arguments, _ := docopt.ParseArgs(usage, nil, appVersion)

	if arguments["--build"] == true {
		fmt.Printf("%v version: %v, built on: %v\n", progName, appVersion, buildTime)
		os.Exit(0)
	}

	consulBool = false
	if arguments["--consul"] == true {
		consulBool = true
	}
	verbose = false
	if arguments["--verbose"] == true {
		verbose = true
	}

	listenPort := arguments["--listen-port"].(string)

	dnsPort = arguments["--dns-port"].(string)
	consulPort = arguments["--consul-port"].(string)
	dnsRecord = arguments["--dns-record"].(string)
	consulRecord = arguments["--consul-record"].(string)

	http.HandleFunc("/ipv4", ipv4)
	http.HandleFunc("/ipv6", ipv6) // IPv6 can be left on by default. If not needed it won't be used.

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%v", listenPort), nil))

}
