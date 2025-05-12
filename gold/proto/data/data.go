package data

import (
	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

func init() {
	link.RegisterDispacher(head.ProtoData, func(header *head.Packet, peer *link.Link, data []byte) {
		peer.ToLower(header, data)
	})
}
