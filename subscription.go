package anycable

type Subscription struct {
	c          *Client
	Identifier ChannelIdentifier
	Subscribed bool
	Messages   chan Event
}
