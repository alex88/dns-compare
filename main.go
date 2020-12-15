package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"sort"
	"strings"

	arg "github.com/alexflint/go-arg"
	"github.com/miekg/dns"
)

var args struct {
	CSV    string   `arg:"positional,required"`
	Server []string `arg:"positional,required"`
}

type ServerResults struct {
	Server string
	Result string
}

func processLine(client *dns.Client, hostname string, recordType string) {
	var questionType uint16
	switch strings.ToUpper(recordType) {
	case "A":
		questionType = dns.TypeA
	case "MX":
		questionType = dns.TypeMX
	case "NS":
		questionType = dns.TypeNS
	case "TXT":
		questionType = dns.TypeTXT
	case "CNAME":
		questionType = dns.TypeCNAME
	case "SRV":
		questionType = dns.TypeSRV
	default:
		fmt.Printf("Invalid DNS record type %s", recordType)
		os.Exit(1)
	}
	q := new(dns.Msg)
	q.SetQuestion(dns.Fqdn(hostname), questionType)

	results := make([]ServerResults, len(args.Server))
	for i, server := range args.Server {
		in, _, err := client.Exchange(q, server)
		if err != nil {
			fmt.Printf("\nError querying server %s for record %s %s: %s", server, recordType, hostname, err)
			os.Exit(1)
		}
		serverResults := make([]string, len(in.Answer))
		for i, a := range in.Answer {
			switch r := a.(type) {
			case *dns.A:
				serverResults[i] = r.A.String()
			case *dns.MX:
				serverResults[i] = fmt.Sprintf("%d,%s", r.Preference, r.Mx)
			case *dns.NS:
				serverResults[i] = r.Ns
			case *dns.CNAME:
				serverResults[i] = r.Target
			case *dns.SRV:
				serverResults[i] = fmt.Sprintf("%d,%d,%d,%s", r.Port, r.Priority, r.Weight, r.Target)
			case *dns.TXT:
				sort.Strings(r.Txt)
				serverResults[i] = strings.Join(r.Txt, "\n")
			default:
				fmt.Printf("Invalid DNS record answer %s", a.String())
				os.Exit(1)
			}
		}
		sort.Strings(serverResults)
		results[i] = ServerResults{Server: server, Result: strings.Join(serverResults, "\n")}
	}
	for _, v := range results[1:] {
		if v.Result != results[0].Result {
			fmt.Printf("\nError\nServer %s returned:\n%s\nServer %s returned:\n%s\n", results[0].Server, results[0].Result, v.Server, v.Result)
			os.Exit(1)
		}
		print("OK")
	}
}

func main() {
	params := arg.MustParse(&args)
	if len(args.Server) < 2 {
		params.Fail("you must provide at least 2 servers")
	}

	for i, s := range args.Server {
		if !strings.Contains(s, ":") {
			args.Server[i] = fmt.Sprintf("%s:53", s)
		}
	}

	file, err := os.Open("input.csv")
	if err != nil {
		fmt.Println("Unable to open input file")
		os.Exit(1)
	}
	defer file.Close()

	client := new(dns.Client)

	r := csv.NewReader(file)
	i := 0
	for {
		i++
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		if len(record) < 2 {
			fmt.Printf("Row %d is invalid", i)
			continue
		}
		fmt.Printf("Processing %s %s... ", record[1], record[0])
		processLine(client, record[0], record[1])
		println()
	}
	println("Done")
}
