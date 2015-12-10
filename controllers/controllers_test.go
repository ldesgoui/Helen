// Copyright (C) 2015  TF2Stadium
// Use of this source code is governed by the GPLv3
// that can be found in the COPYING file.

package controllers_test

import (
	"io/ioutil"
	"math/rand"
	"net/url"
	"strconv"
	"sync"
	"testing"

	db "github.com/TF2Stadium/Helen/database"
	"github.com/TF2Stadium/Helen/helpers"
	"github.com/TF2Stadium/Helen/models"
	"github.com/TF2Stadium/Helen/testhelpers"
	"github.com/TF2Stadium/wsevent"
	"github.com/stretchr/testify/assert"
)

func init() {
	helpers.InitLogger()
	testhelpers.CleanupDB()
}

func TestLogin(t *testing.T) {
	server := testhelpers.StartServer(wsevent.NewServer(), wsevent.NewServer())
	defer server.Close()

	var count int

	steamid := strconv.Itoa(rand.Int())
	client := testhelpers.NewClient()

	resp, err := testhelpers.Login(steamid, client)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	bytes, _ := ioutil.ReadAll(resp.Body)
	t.Log(string(bytes))

	player, tperr := models.GetPlayerBySteamId(steamid)
	assert.NoError(t, tperr)
	assert.NotNil(t, player)
	assert.Equal(t, player.SteamId, steamid)

	assert.Nil(t, db.DB.Table("http_sessions").Count(&count).Error)
	assert.NotEqual(t, count, 0)
	assert.NotNil(t, client.Jar)
}

func TestWS(t *testing.T) {
	server := testhelpers.StartServer(wsevent.NewServer(), wsevent.NewServer())
	defer server.Close()

	steamid := strconv.Itoa(rand.Int())
	client := testhelpers.NewClient()

	resp, err := testhelpers.Login(steamid, client)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	addr, _ := url.Parse("http://localhost:8080/")
	client.Jar.Cookies(addr)
	conn, err := testhelpers.ConnectWS(client)
	assert.NoError(t, err)
	assert.NotNil(t, conn)

	testhelpers.ReadMessages(conn, testhelpers.InitMessages, t)
}

func BenchmarkWS(b *testing.B) {
	server := testhelpers.StartServer(wsevent.NewServer(), wsevent.NewServer())
	defer server.Close()
	wg := new(sync.WaitGroup)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		go func() {
			wg.Add(1)
			defer wg.Done()

			steamid := strconv.Itoa(rand.Int())
			client := testhelpers.NewClient()
			_, err := testhelpers.Login(steamid, client)
			if err != nil {
				b.Error(err)
				b.FailNow()
			}

			conn, err := testhelpers.ConnectWS(client)
			if err != nil {
				b.Error(err)
				b.FailNow()
			}

			for i := 0; i < 5; i++ {
				_, _, err := conn.ReadMessage()
				if err != nil {
					b.Error(err)
					b.FailNow()
				}
			}
		}()
	}
	wg.Wait()
}

func TestSocketInfo(t *testing.T) {
	server := testhelpers.StartServer(testhelpers.NewSockets())
	defer server.Close()

	client := testhelpers.NewClient()
	testhelpers.Login(strconv.Itoa(rand.Int()), client)
	conn, err := testhelpers.ConnectWS(client)
	defer conn.Close()
	assert.NoError(t, err)

	_, err = testhelpers.ReadMessages(conn, testhelpers.InitMessages, nil)
	assert.NoError(t, err)

	args := map[string]interface{}{
		"id": "1",
		"data": map[string]interface{}{
			"request": "getSocketInfo",
		},
	}

	reply, err := testhelpers.EmitJSONWithReply(conn, args)
	assert.NoError(t, err)
	//assert.Equal(t, reply["rooms"].([]interface{})[0].(string), "0_public")
	t.Logf("%v", reply)

}

func TestLobbyCreate(t *testing.T) {
	server := testhelpers.StartServer(testhelpers.NewSockets())
	defer server.Close()

	steamid := strconv.Itoa(rand.Int())
	client := testhelpers.NewClient()
	testhelpers.Login(steamid, client)
	conn, err := testhelpers.ConnectWS(client)
	defer conn.Close()
	assert.NoError(t, err)
	_, err = testhelpers.ReadMessages(conn, testhelpers.InitMessages, nil)
	assert.NoError(t, err)

	args := map[string]interface{}{
		"id": "1",
		"data": map[string]interface{}{
			"request":        "lobbyCreate",
			"map":            "cp_badlands",
			"type":           "6s",
			"league":         "etf2l",
			"server":         "testerino",
			"rconpwd":        "testerino",
			"whitelistID":    123,
			"mumbleRequired": true,
		}}

	reply, err := testhelpers.EmitJSONWithReply(conn, args)
	assert.NoError(t, err)
	assert.True(t, reply["success"].(bool))
	id := uint(reply["data"].(map[string]interface{})["id"].(float64))
	t.Logf("%v", reply)

	lobby, err := models.GetLobbyById(id)
	assert.NoError(t, err)
	assert.Equal(t, lobby.CreatedBySteamID, steamid)
}

func TestLobbyJoin(t *testing.T) {
	server := testhelpers.StartServer(testhelpers.NewSockets())
	defer server.Close()

	steamid := strconv.Itoa(rand.Int())
	client := testhelpers.NewClient()
	testhelpers.Login(steamid, client)
	player, tperr := models.GetPlayerBySteamId(steamid)
	assert.NoError(t, tperr)
	conn, err := testhelpers.ConnectWS(client)
	defer conn.Close()
	assert.NoError(t, err)
	_, err = testhelpers.ReadMessages(conn, testhelpers.InitMessages, nil)
	assert.NoError(t, err)

	args := map[string]interface{}{
		"id": "1",
		"data": map[string]interface{}{
			"request": "lobbyJoin",
			"id":      1,
			"class":   "scout1",
			"team":    "red",
		}}

	conn.WriteJSON(args)

	assert.Equal(t, testhelpers.ReadJSON(conn)["request"].(string), "lobbyJoined")
	assert.True(t, testhelpers.ReadJSON(conn)["success"].(bool))
	id, tperr := player.GetLobbyId()
	assert.NoError(t, tperr)
	if id != 1 {
		t.Fatal("Got wrong ID")
	}

	lobby, tperr := models.GetLobbyById(1)
	assert.NoError(t, tperr)
	assert.Equal(t, lobby.GetPlayerNumber(), 1)
}

func TestSpectatorJoin(t *testing.T) {
	server := testhelpers.StartServer(testhelpers.NewSockets())
	defer server.Close()

	steamid := strconv.Itoa(rand.Int())
	client := testhelpers.NewClient()
	testhelpers.Login(steamid, client)
	conn, err := testhelpers.ConnectWS(client)
	defer conn.Close()
	assert.NoError(t, err)
	_, err = testhelpers.ReadMessages(conn, testhelpers.InitMessages, nil)
	assert.NoError(t, err)

	conn.WriteJSON(
		map[string]interface{}{
			"id": "1",
			"data": map[string]interface{}{
				"request": "lobbySpectatorJoin",
				"id":      1,
			},
		})
	testhelpers.ReadMessages(conn, 1, nil)
	assert.True(t, testhelpers.ReadJSON(conn)["success"].(bool))

	//Send ChatMessages
	conn.WriteJSON(
		map[string]interface{}{
			"id": "1",
			"data": map[string]interface{}{
				"request": "getSocketInfo",
			},
		})

	recv := testhelpers.ReadJSON(conn)
	assert.Equal(t, len(recv["rooms"].([]interface{})), 2)
}

func TestChatSend(t *testing.T) {
	server := testhelpers.StartServer(testhelpers.NewSockets())
	defer server.Close()

	steamid := strconv.Itoa(rand.Int())
	client := testhelpers.NewClient()
	testhelpers.Login(steamid, client)
	conn, err := testhelpers.ConnectWS(client)
	defer conn.Close()
	assert.NoError(t, err)
	_, err = testhelpers.ReadMessages(conn, testhelpers.InitMessages, nil)
	assert.NoError(t, err)

	conn.WriteJSON(
		map[string]interface{}{
			"id": "1",
			"data": map[string]interface{}{
				"request": "chatSend",
				"message": "testerino",
				"room":    0,
			},
		})

	assert.True(t, testhelpers.ReadJSON(conn)["success"].(bool))

	recv := testhelpers.ReadJSON(conn)
	assert.Equal(t, recv["data"].(map[string]interface{})["player"].(map[string]interface{})["steamid"], steamid)
}
