package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/monochromegane/slack-incoming-webhooks"
	"github.com/mackerelio/mackerel-agent/logging"
	"github.com/mackerelio/mackerel-client-go"
)

var logger = logging.GetLogger("main")

var mackerelBase = "https://mackerel.io"

var pollingInterval = 20 * time.Second

func main() {
	os.Exit(run())
}

func run() int {
	apiKey := os.Getenv("MACKEREL_APIKEY")
	if apiKey == "" {
		logger.Errorf(`MACKEREL_APIKEY environment variable is not set. (Try "export MACKEREL_APIKEY='<Your apikey>'"`)
		return 1
	}
	slackURL := os.Getenv("MKRP_SLACK_WEBHOOK_URL")
	if slackURL == "" {
		logger.Errorf(`MKRP_SLACK_WEBHOOK_URL environment variable is not set. (Try "export MKRP_SLACK_WEBHOOK_URL='<Your apikey>'"`)
		return 1
	}
	slackChannel := os.Getenv("MKRP_SLACK_CHANNEL") // optional

	s := &slack{
		client: slack_incoming_webhooks.Client{
			WebhookURL: slackURL,
		},
		channel:    slackChannel,
	}

	a := &app{
		cli:   mackerel.NewClient(apiKey),
		slack: s,
	}
	return a.loop()
}

type app struct {
	cli   *mackerel.Client
	org   string
	hosts map[string]*mackerel.Host
	slack *slack
}

func (a *app) getOrg() string {
	if a.org == "" {
		org, _ := a.cli.GetOrg()
		a.org = org.Name
	}
	return a.org
}

func (a *app) getHosts() (map[string](*mackerel.Host), error) {
	hosts, err := a.cli.FindHosts(&mackerel.FindHostsParam{
		Statuses: []string{"standby", "working", "maintenance", "poweroff"},
	})
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
				hds.org = a.getOrg()
				if hds.hasDiff() {
					a.slack.post(hds)
					fmt.Println(hds.String())
				}
			}
		}
		time.Sleep(pollingInterval)
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
	org          string
	newHosts     []*mackerel.Host
	retiredHosts []*mackerel.Host
	changedHosts []*changedHost
}

func (hds *hostDiffs) hasDiff() bool {
	return (len(hds.newHosts) + len(hds.retiredHosts) + len(hds.changedHosts)) > 0
}

func (hds *hostDiffs) String() string {
	s := ""
	if len(hds.newHosts) > 0 {
		s += "New:\n"
	}
	for _, h := range hds.newHosts {
		s += "- " + formatHost(hds.org, h) + "\n"
	}
	if len(hds.retiredHosts) > 0 {
		s += "Retired:\n"
	}
	for _, h := range hds.retiredHosts {
		s += "- " + formatHost(hds.org, h) + "\n"
	}
	if len(hds.changedHosts) > 0 {
		s += "Changed:\n"
	}
	for _, h := range hds.changedHosts {
		s += "- " + formatHost(hds.org, h.host) + "\n"
		if h.statusChanged() {
			s += fmt.Sprintf("    status: %s -> %s\n", h.oldStatus, h.host.Status)
		}
		if h.roleChanged() {
			s += fmt.Sprintf("    roles: %s -> %s\n", strings.Join(h.oldRoles, ","), strings.Join(h.host.GetRoleFullnames(), ","))
		}
	}
	return s
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

func formatHost(org string, h *mackerel.Host) string {
	return fmt.Sprintf("%s status: %s, roles: %s, %s",
		formatHostName(h), h.Status, strings.Join(h.GetRoleFullnames(), ","), formatHostURL(org, h.ID))
}

func formatHostURL(org, id string) string {
	return fmt.Sprintf("%s/orgs/%s/hosts/%s", mackerelBase, org, id)
}

func formatRoleURL(org, roleFullname string) string {
	s := strings.Split(roleFullname, ":")
	service := s[0]
	role := s[1]
	return fmt.Sprintf("%s/orgs/%s/services/%s/%s/-/graph", mackerelBase, service, role)
}

func formatHostName(h *mackerel.Host) string {
	s := h.Name
	if h.DisplayName != "" {
		s += fmt.Sprintf("(%s)", h.DisplayName)
	}
	return s
}
