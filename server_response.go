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
	case PacketTypeJoinLobbyRequest:
		s.HandleJoinLobbyRequest(client, packet.(JoinLobbyRequestPacket))
	case PacketTypeLeaveLobbyRequest:
		s.HandleLeaveLobbyRequest(client, packet.(LeaveLobbyRequestPacket))
	}
}

func (s *Server) HandleVersionRequest(client net.Conn, packet VersionPacket) {
	debugString := fmt.Sprintf("HandleVersionPacket: %+v", packet)
	s.Log(debugString)

	if packet.Version != ProtocolVersion {
		if s.Debug {
			logEntry := fmt.Sprintf("Client %s sent a version packet with a mismatching version.", client.RemoteAddr())
			s.Log(logEntry)
		}

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
			s.Log(logEntry)
		}

		return e
	}

	player.Lock()
	defer player.Unlock()

	lobby := s.AddLobby(player, packet.Name)

	logEntry := fmt.Sprintf("Player %s created lobby %s (%s)", player.Name, lobby.name, lobby.ID())
	s.Log(logEntry)

	e = s.SendCreateLobbyResponse(player, lobby.ID())
	return e
}

func (s *Server) HandlePlayerListRequest(client net.Conn, packet PlayerListRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		if s.Debug {
			logEntry := fmt.Sprintf("HandlePlayerListRequest: %s", e.Error())
			s.Log(logEntry)
		}
		return e
	}

	player.Lock()
	defer player.Unlock()

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
	s.Log(logEntry)
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

	player.Lock()
	playerName := player.Name
	player.Unlock()

	for _, p := range s.Clients {
		p.Lock()
		defer p.Unlock()
		s.SendGlobalChatResponse(p, playerName, packet.Message)
	}

	logEntry := fmt.Sprintf("(Global Chat) %s: %s", playerName, packet.Message)
	s.Log(logEntry)
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
	s.Log(logEntry)
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

		for _, p := range lobby.Players() {
			p.Lock()
			defer p.Unlock()
			s.SendJoinLobbyResponse(p, lobby.name, lobby.ID(), true)
		}

		return nil
	}

	lobby, e = s.FindLobbyByName(packet.Name)

	if e == nil {
		lobby.AddPlayer(player)

		for _, p := range lobby.Players() {
			p.Lock()
			defer p.Unlock()
			s.SendJoinLobbyResponse(p, lobby.name, lobby.ID(), true)
		}

		return nil
	}

	return errors.New("no such lobby found")
}

func (s *Server) HandleLeaveLobbyRequest(client net.Conn, packet LeaveLobbyRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		return e
	}

	if !s.PlayerHasLobby(player) {
		return errors.New("player is not in a lobby")
	}

	lobby, e := s.FindLobbyByID(packet.ID)

	if e == nil {
		lobby.RemovePlayer(player)

		for _, p := range lobby.Players() {
			s.SendLeaveLobbyResponse(p, lobby.name, lobby.ID(), true)
		}

		return nil
	}

	lobby, e = s.FindLobbyByName(packet.Name)

	if e == nil {
		lobby.RemovePlayer(player)

		for _, p := range lobby.Players() {
			p.Lock()
			defer p.Unlock()
			s.SendLeaveLobbyResponse(p, lobby.name, lobby.ID(), true)
		}

		return nil
	}

	return errors.New("no such lobby found")
}

func (s *Server) HandleLobbyKickRequest(client net.Conn, packet LobbyKickRequestPacket) error {
	player, e := s.FindPlayerByConnection(client)

	if e != nil {
		return e
	}

	targetPlayer, e := s.FindPlayerByID(packet.Target)

	if e != nil {
		return e
	}

	lobby, e := s.FindLobbyByPlayer(player)

	if e != nil {
		return e
	}

	if lobby.Host() != player {
		logEntry := fmt.Sprintf("%s attempted to kick player %s, but they are not the host.", player.Name, targetPlayer.Name)
		s.Log(logEntry)
		return errors.New("insufficient privileges; must be lobby host to kick users")
	}

	lobby.RemovePlayer(targetPlayer)
	s.SendLobbyKickResponse(player, targetPlayer.ID, true)
	s.SendLobbyKickResponse(targetPlayer, targetPlayer.ID, true)

	return nil
}
