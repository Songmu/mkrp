package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mackerelio/mackerel-agent/logging"
	"github.com/mackerelio/mackerel-client-go"
)

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
	return a.loop()
}

type app struct {
	cli   *mackerel.Client
	hosts map[string]*mackerel.Host
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

func (a *app) loop() int {
	for {
		hosts, err := a.getHosts()
		if err == nil {
			oldHosts := a.hosts
			a.hosts = hosts
			if oldHosts != nil {
				hds := getHostDiffs(oldHosts, hosts)
				fmt.Printf("%+v\n", hds)
			}
		}
		time.Sleep(20 * time.Second)
	}
	return 0
}

type changedHost struct {
	host      *mackerel.Host
	oldStatus string
	oldRoles  []string
}

func (c *changedHost) roleChanged() bool {
	return c.oldRoles != nil
}

func (c *changedHost) statusChanged() bool {
	return c.oldStatus != ""
}

type hostDiffs struct {
	newHosts     []*mackerel.Host
	retiredHosts []*mackerel.Host
	changedHosts []*changedHost
}

func getHostDiffs(old, new map[string]*mackerel.Host) *hostDiffs {
	hds := &hostDiffs{}
	for k, v := range old {
		if _, ok := new[k]; !ok {
			hds.retiredHosts = append(hds.retiredHosts, v)
		}
	}

	for k, v := range new {
		oldHost, ok := old[k]
		if !ok {
			hds.newHosts = append(hds.newHosts, v)
		} else {
			d := getHostDiff(oldHost, v)
			if d != nil {
				hds.changedHosts = append(hds.changedHosts, d)
			}
		}
	}
	return hds
}

func getHostDiff(old, new *mackerel.Host) *changedHost {
	c := &changedHost{
		host: new,
	}
	if old.Status != new.Status {
		c.oldStatus = old.Status
	}

	oldRoles := old.GetRoleFullnames()
	if len(oldRoles) == 0 {
		oldRoles = []string{}
	}
	sort.Strings(oldRoles)
	newRoles := new.GetRoleFullnames()
	if len(newRoles) == 0 {
		newRoles = []string{}
	}
	sort.Strings(newRoles)
	if strings.Join(oldRoles, ",") != strings.Join(newRoles, ",") {
		c.oldRoles = oldRoles
	}

	if c.roleChanged() || c.statusChanged() {
		return c
	}
	return nil
}
