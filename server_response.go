package kfnetwork

import (
	"errors"
	"fmt"
	"net"
)

func (s *Server) HandlePacket(client net.Conn, packet Packet) {
	switch packet.GetHeader().Type {
	case PacketTypeVersionRequest:
		s.HandleVersionRequest(client, packet.(VersionPacket))
	case PacketTypeLoginRequest:
		s.HandleLoginRequest(client, packet.(LoginRequestPacket))
	case PacketTypeGlobalChatRequest:
		s.HandleGlobalChatRequest(client, packet.(GlobalChatRequestPacket))
	case PacketTypePlayerListRequest:
		s.HandlePlayerListRequest(client, packet.(PlayerListRequestPacket))
	case PacketTypeCreateLobbyRequest:
		s.HandleCreateLobbyRequest(client, packet.(CreateLobbyRequestPacket))
	case PacketTypeLobbyListRequest:
		s.HandleLobbyListRequest(client, packet.(LobbyListRequestPacket))
	}
}

func (s *Server) HandleVersionRequest(client net.Conn, packet VersionPacket) {
	if packet.Version != ProtocolVersion {
		logEntry := fmt.Sprintf("Client %s sent a version packet with a mismatching version.", client.RemoteAddr())
		Logger().Error(logEntry)
		s.SendErrorPacket(client, "Protocol version mismatch.")
		s.CloseConnection(client)
	}
}

func (s *Server) HandleLoginRequest(client net.Conn, packet LoginRequestPacket) error {
	vaultUser, e := RetrieveProfile(packet.Token)

	if e != nil {
		return e
	}

	if vaultUser.ID != packet.ID {
		s.SendErrorPacket(client, "Login failed.")
		s.CloseConnection(client)
		return errors.New("incorrect user ID supplied in packet")
	}

	player := NewPlayer()
	player.Name = packet.Name
	player.ID = packet.ID
	player.Client = client

	s.AddPlayer(player)
	return nil
}

func (s *Server) HandleExitRequest(client net.Conn, packet ExitPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		return e
	}

	s.CloseConnection(player.Client)
	s.RemovePlayer(player)
	return nil
}

func (s *Server) HandleCreateLobbyRequest(client net.Conn, packet CreateLobbyRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		if s.Debug {
			logEntry := fmt.Sprintf("HandleCreateLobbyRequest: %s", e.Error())
			Logger().Log(logEntry)
		}

		return e
	}

	lobby := s.AddLobby(player, packet.Name)

	logEntry := fmt.Sprintf("Player %s created lobby %s (%s)", player.Name, lobby.name, lobby.ID())
	Logger().Log(logEntry)

	e = s.SendCreateLobbyResponse(player, lobby.ID())
	return e
}

func (s *Server) HandlePlayerListRequest(client net.Conn, packet PlayerListRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		if s.Debug {
			logEntry := fmt.Sprintf("HandlePlayerListRequest: %s", e.Error())
			Logger().Log(logEntry)
		}
		return e
	}

	playerList := PlayerList{}

	for _, p := range s.Clients {
		entry := PlayerListEntry{}
		entry.ID = p.ID
		entry.Name = p.Name

		playerList.Players = append(playerList.Players, entry)
	}

	playerList.Count = uint(len(playerList.Players))

	e = s.SendPlayerListResponse(player, playerList)

	if e != nil {
		return e
	}

	logEntry := fmt.Sprintf("Player %s requested the player list", player.Name)
	Logger().Log(logEntry)
	return nil
}

func (s *Server) HandleLobbyChatRequest(client net.Conn, packet LobbyChatRequestPacket) error {
	return nil
}

func (s *Server) HandleGlobalChatRequest(client net.Conn, packet GlobalChatRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		return e
	}

	for _, p := range s.Clients {
		s.SendGlobalChatResponse(p, player.Name, packet.Message)
	}

	logEntry := fmt.Sprintf("(Global Chat) %s: %s", player.Name, packet.Message)
	Logger().Log(logEntry)
	return nil
}

func (s *Server) HandleLobbyListRequest(client net.Conn, packet LobbyListRequestPacket) error {
	lobbyList := LobbyList{}

	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		return e
	}

	for _, lobby := range s.Lobbies {
		entry := LobbyListEntry{ID: lobby.ID(), Name: lobby.Name()}
		lobbyList.Lobbies = append(lobbyList.Lobbies, entry)
	}

	lobbyList.Count = uint(len(lobbyList.Lobbies))

	s.SendLobbyListResponse(player, lobbyList)

	logEntry := fmt.Sprintf("Player %s requested a lobby list.", player.Name)
	Logger().Log(logEntry)
	return nil
}

func (s *Server) HandleJoinLobbyRequest(client net.Conn, packet JoinLobbyRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		return e
	}

	lobby, e := s.FindLobbyByID(packet.ID)

	if e == nil {
		lobby.AddPlayer(player)
		s.SendJoinLobbyResponse(player, lobby.name, lobby.ID(), true)
		return nil
	}

	lobby, e = s.FindLobbyByName(packet.Name)

	if e == nil {
		lobby.AddPlayer(player)
		s.SendJoinLobbyResponse(player, lobby.name, lobby.ID(), true)
		return nil
	}

	return errors.New("no lobby found")
}
