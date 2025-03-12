package data

import (
	"github.com/fumiama/orbyte/pbuf"

	"github.com/fumiama/WireGold/gold/head"
	"github.com/fumiama/WireGold/gold/link"
)

func init() {
	link.RegisterDispacher(head.ProtoData, func(header *head.Packet, peer *link.Link, data pbuf.Bytes) {
		peer.ToLower(header, data)
	})
}
