// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package socket

import (
	"encoding/json"
	"errors"

	"github.com/TF2Stadium/Helen/controllers/broadcaster"
	chelpers "github.com/TF2Stadium/Helen/controllers/controllerhelpers"
	"github.com/TF2Stadium/Helen/controllers/controllerhelpers/hooks"
	"github.com/TF2Stadium/Helen/controllers/socket/internal/handler"
	"github.com/TF2Stadium/Helen/helpers"
	"github.com/TF2Stadium/Helen/models"
	"github.com/TF2Stadium/wsevent"
)

var ErrRecordNotFound = errors.New("Player record for found.")

func getEvent(data []byte) string {
	var js struct {
		Request string
	}
	json.Unmarshal(data, &js)
	return js.Request
}

func ServerInit(server *wsevent.Server, noAuthServer *wsevent.Server) {
	server.OnDisconnect = hooks.OnDisconnect
	server.Extractor = getEvent

	noAuthServer.OnDisconnect = hooks.OnDisconnect
	noAuthServer.Extractor = getEvent

	server.On("authenticationTest", func(server *wsevent.Server, so *wsevent.Client, data []byte) interface{} {
		return struct {
			Message string `json:"message"`
		}{"authenticated"}
	})
	//Global Handlers
	server.Register(handler.Global{})
	//Lobby Handlers
	server.Register(handler.Lobby{})
	//server.On("lobbyCreate", handler.LobbyCreate)
	//Player Handlers
	server.Register(handler.Player{})
	//Chat Handlers
	server.Register(handler.Chat{})
	//Admin Handlers
	server.Register(handler.Admin{})
	//Ban Handlers
	handler.InitializeBans(server)
	//Debugging handlers
	// if config.Constants.ServerMockUp {
	// 	server.On("debugLobbyFill", handler.DebugLobbyFill)
	// 	server.On("debugLobbyReady", handler.DebugLobbyReady)
	// 	server.On("debugUpdateStatsFilter", handler.DebugUpdateStatsFilter)
	// 	server.On("debugPlayerSub", handler.DebugPlayerSub)
	// }

	server.DefaultHandler = func(_ *wsevent.Server, _ *wsevent.Client, _ []byte) interface{} {
		return helpers.NewTPError("No such request.", -3)
	}

	noAuthServer.On("lobbySpectatorJoin", func(s *wsevent.Server, so *wsevent.Client, data []byte) interface{} {
		var args struct {
			Id *uint `json:"id"`
		}

		if err := chelpers.GetParams(data, &args); err != nil {
			return helpers.NewTPErrorFromError(err)
		}

		var lob *models.Lobby
		lob, tperr := models.GetLobbyByID(*args.Id)

		if tperr != nil {
			return tperr
		}

		hooks.AfterLobbySpec(s, so, lob)

		so.EmitJSON(helpers.NewRequest("lobbyData", models.DecorateLobbyData(lob, true)))

		return chelpers.EmptySuccessJS
	})
	noAuthServer.On("getSocketInfo", (handler.Global{}).GetSocketInfo)

	noAuthServer.DefaultHandler = func(_ *wsevent.Server, so *wsevent.Client, data []byte) interface{} {
		return helpers.NewTPError("Player isn't logged in.", -4)
	}
}

func SocketInit(server *wsevent.Server, noauth *wsevent.Server, so *wsevent.Client) error {
	chelpers.AuthenticateSocket(so.Id(), so.Request())
	loggedIn := chelpers.IsLoggedInSocket(so.Id())
	if loggedIn {
		steamid := chelpers.GetSteamId(so.Id())
		broadcaster.SetSocket(steamid, so)
	}

	if loggedIn {
		hooks.AfterConnect(server, so)

		player, err := models.GetPlayerBySteamID(chelpers.GetSteamId(so.Id()))
		if err != nil {
			helpers.Logger.Warning(
				"User has a cookie with but a matching player record doesn't exist: %s",
				chelpers.GetSteamId(so.Id()))
			chelpers.DeauthenticateSocket(so.Id())
			hooks.AfterConnect(noauth, so)
			return ErrRecordNotFound
		}

		hooks.AfterConnectLoggedIn(server, so, player)
	} else {
		hooks.AfterConnect(noauth, so)
		so.EmitJSON(helpers.NewRequest("playerSettings", "{}"))
		so.EmitJSON(helpers.NewRequest("playerProfile", "{}"))
	}

	so.EmitJSON(helpers.NewRequest("socketInitialized", "{}"))

	return nil
}
