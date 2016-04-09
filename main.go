package main

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/mackerelio/mackerel-agent/logging"
	"github.com/mackerelio/mackerel-client-go"
	"github.com/sergi/go-diff/diffmatchpatch"
)

// new host
// retired host
// changed status
// role changed

var logger = logging.GetLogger("main")

func main() {
	os.Exit(run())
}

func run() int {
	apiKey := os.Getenv("MACKEREL_APIKEY")
	if apiKey == "" {
		logger.Errorf(`MACKEREL_APIKEY environment variable is not set. (Try "export MACKEREL_APIKEY='<Your apikey>'"`)
		return 1
	}
	a := &app{
		cli: mackerel.NewClient(apiKey),
	}
	hosts, _ := a.getHosts()

	fmt.Printf("%+v\n", hosts)

	return 0
}

type app struct {
	cli *mackerel.Client
}

func (a *app) getHosts() (map[string](*mackerel.Host), error) {
	hosts, err := a.cli.FindHosts(&mackerel.FindHostsParam{})
	if err != nil {
		return nil, err
	}
	ret := make(map[string](*mackerel.Host))
	for _, h := range hosts {
		ret[h.ID] = h
	}
	return ret, nil
}

type diffs struct {
	added   []string
	deleted []string
}

func sliceDiff(old, new []string) diffs {
	sort.Strings(old)
	sort.Strings(new)
	delim := "\n"
	dmp := diffmatchpatch.New()
	a, b, c := dmp.DiffLinesToChars(strings.Join(old, delim), strings.Join(new, delim))
	diff := dmp.DiffCharsToLines(dmp.DiffMain(a, b, false), c)
	d := diffs{}
	for _, v := range diff {
		switch v.Type {
		case diffmatchpatch.DiffInsert:
			elms := strings.Split(strings.TrimSpace(v.Text), delim)
			d.added = append(d.added, elms...)
		case diffmatchpatch.DiffDelete:
			elms := strings.Split(strings.TrimSpace(v.Text), delim)
			d.deleted = append(d.deleted, elms...)
		}
	}
	return d
}
