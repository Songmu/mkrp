package main

import "github.com/monochromegane/slack-incoming-webhooks"

type slack struct {
	client  slack_incoming_webhooks.Client
	channel string
}

func (s *slack) post(hds *hostDiffs) {
	s.client.Post(&slack_incoming_webhooks.Payload{
		Text: hds.String(),
		Channel: s.channel,
		IconURL: "https://dl.dropboxusercontent.com/u/289339/mackerel/agent-si.png",
		Username: "Mackerel Host information",
	})
}
