package kfnetwork

import (
	"net"
)

type Client struct {
	Connection net.Conn
	Sequence   uint16
}

func NewClient() *Client {
	client := new(Client)
	return client
}

func (c *Client) Connect(address string) error {
	var e error
	c.Connection, e = net.Dial("tcp", address)

	if e != nil {
		return e
	}

	return nil
}

func (c *Client) SendVersionRequest() error {
	packet := VersionPacket{}
	packet.Type = PacketTypeVersionRequest
	packet.Version = ProtocolVersion

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendExitRequest() error {
	packet := ExitPacket{}
	packet.Type = PacketTypeExit

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendLoginRequest(name string, id string, token string) error {
	packet := LoginRequestPacket{}
	packet.Type = PacketTypeLoginRequest
	packet.Name = name
	packet.ID = id
	packet.Token = token

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendCreateLobbyRequest(name string) error {
	packet := CreateLobbyRequestPacket{}
	packet.Type = PacketTypeCreateLobbyRequest
	packet.Name = name

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendGetCardPile(pile uint8) error {
	packet := CardPileRequestPacket{}
	packet.Type = PacketTypeCardPileRequest
	packet.Pile = pile

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendPlayerListRequest() error {
	packet := PlayerListRequestPacket{}
	packet.Type = PacketTypePlayerListRequest

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendGlobalChatRequest(message string) error {
	packet := GlobalChatRequestPacket{}
	packet.Type = PacketTypeGlobalChatRequest
	packet.Message = message

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendLobbyListRequest() error {
	packet := LobbyListRequestPacket{}
	packet.Type = PacketTypeLobbyListRequest

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendJoinLobbyRequest(query string) error {
	packet := JoinLobbyRequestPacket{}
	packet.Type = PacketTypeJoinLobbyRequest
	packet.Name = query

	e := WritePacket(c.Connection, packet)
	c.Sequence++
	return e
}

func (c *Client) SendGetArchivePile() {
	c.SendGetCardPile(CardPileArchive)
}
