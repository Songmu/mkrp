package main

import (
	"fmt"
	"strings"

	"github.com/mackerelio/mackerel-client-go"
	"github.com/monochromegane/slack-incoming-webhooks"
)

type slack struct {
	client  slack_incoming_webhooks.Client
	channel string
}

func (s *slack) post(hds *hostDiffs) {
	s.client.Post(&slack_incoming_webhooks.Payload{
		Attachments: hds.slackAttachments(),
		Channel:  s.channel,
		IconURL:  "https://raw.githubusercontent.com/Songmu/mkrp/master/_assets/agent-si.png",
		Username: "Mackerel Host information",
	})
}

func (hds *hostDiffs) slackAttachments() (ats []*slack_incoming_webhooks.Attachment) {
	if len(hds.newHosts) > 0 {
		at := &slack_incoming_webhooks.Attachment{
			Title: "New:",
			Color: "#4dbddb",
		}
		s := ""
		for _, h := range hds.newHosts {
			s += "- " + formatHostForSlack(hds.org, h) + "\n"
		}
		at.Text = s
		ats = append(ats, at)
	}
	if len(hds.retiredHosts) > 0 {
		at := &slack_incoming_webhooks.Attachment{
			Title: "Retired:",
			Color: "#c6cacc",
		}
		s := ""
		for _, h := range hds.retiredHosts {
			s += "- " + formatHostForSlack(hds.org, h) + "\n"
		}
		at.Text = s
		ats = append(ats, at)
	}
	if len(hds.changedHosts) > 0 {
		at := &slack_incoming_webhooks.Attachment{
			Title: "Changed:",
			Color: "#ffcd16",
		}
		s := ""
		for _, h := range hds.changedHosts {
			s += "- " + formatHostForSlack(hds.org, h.host) + "\n"
			if h.statusChanged() {
				s += fmt.Sprintf("    status: %s -> %s\n", h.oldStatus, h.host.Status)
			}
			if h.roleChanged() {
				s += fmt.Sprintf("    roles: %s -> %s\n", strings.Join(h.oldRoles, ","), strings.Join(h.host.GetRoleFullnames(), ","))
			}
		}
		at.Text = s
		ats = append(ats, at)
	}
	return ats
}

func formatHostForSlack(org string, h *mackerel.Host) string {
	return fmt.Sprintf("<%s|%s> status: %s, roles: %s", formatHostURL(org, h.ID), formatHostName(h), h.Status, formatRolesForSlack(org, h.GetRoleFullnames()))
}

func formatRolesForSlack(org string, roleFullnames []string) string {
	ret := []string{}
	for _, r := range roleFullnames {
		p := fmt.Sprintf("<%s|%s>", formatRoleURL(org, r), r)
		ret = append(ret, p)
	}
	return strings.Join(ret, ", ")
}
