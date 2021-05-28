package mainScreen

import (
	"github.com/ruraomsk/TLServer/logger"
	"github.com/ruraomsk/VPUserver/model/accToken"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/VPUserver/model/license"
	"time"
)

type HubMainScreen struct {
	clients    map[*ClientMS]bool
	broadcast  chan mSResponse
	register   chan *ClientMS
	unregister chan *ClientMS
}

func NewMainScreenHub() *HubMainScreen {
	return &HubMainScreen{
		broadcast:  make(chan mSResponse),
		clients:    make(map[*ClientMS]bool),
		register:   make(chan *ClientMS),
		unregister: make(chan *ClientMS),
	}
}

func (h *HubMainScreen) Run() {
	UserLogoutGS = make(chan string, 5)
	data.AccAction = make(chan string, 50)
	license.LogOutAllFromLicense = make(chan bool)
	checkValidityTicker := time.NewTicker(checkTokensValidity)
	phoneReadTick := time.NewTicker(phoneTick)
	defer func() {
		phoneReadTick.Stop()
		checkValidityTicker.Stop()
	}()
	for {
		select {
		case <-phoneReadTick.C:
			{
				h.sendFhones()
			}

		case client := <-h.register:
			{
				flag, tk := checkToken(client.cookie, client.cInfo.IP)
				resp := newMainMess(typeMapInfo, nil)
				if flag {
					resp.Data["role"] = tk.Role
					resp.Data["access"] = data.AccessCheck(tk.Login, 2, 5, 6, 7, 8, 9, 10)
					resp.Data["description"] = tk.Description
					resp.Data["authorizedFlag"] = true
					resp.Data["region"] = tk.Region
					client.cInfo = tk
					client.send <- resp
				}
				h.clients[client] = true
				h.sendFhones()
			}
		case client := <-h.unregister:
			{
				if _, ok := h.clients[client]; ok {
					client.isLogin = false
					//close(client.send)
					//_ = client.conn.Close()
					logger.Debug.Printf("остановили клиента %s", client.cInfo.Login)
				} else {
					logger.Debug.Printf("нет клиента %s", client.cInfo.Login)
				}
			}
		case mess := <-h.broadcast:
			{
				for client := range h.clients {
					select {
					case client.send <- mess:
					default:
						delete(h.clients, client)
						close(client.send)
					}
				}
			}
		case login := <-data.AccAction:
			{
				respLO := newMainMess(typeLogOut, nil)
				status := logOut(login)
				if status {
					respLO.Data["authorizedFlag"] = false
				}
				for client := range h.clients {
					if client.cInfo.Login == login {
						client.send <- respLO
					}
				}
				logOutSockets(login)
			}
		case <-checkValidityTicker.C:
			{
				for client := range h.clients {
					if client.cookie != "" {
						if client.cInfo.Valid() != nil {
							resp := newMainMess(typeLogOut, nil)
							status := logOut(client.cInfo.Login)
							if status {
								resp.Data["authorizedFlag"] = false
							}
							client.cInfo = new(accToken.Token)
							client.cookie = ""
							client.send <- resp
						}
					}
				}
			}
		case <-license.LogOutAllFromLicense:
			{
				for client := range h.clients {
					data.AccAction <- client.cInfo.Login
				}
			}
		case login := <-UserLogoutGS:
			{
				for client := range h.clients {
					if client.cInfo.Login == login {
						if _, ok := h.clients[client]; ok {
							client.isLogin = false
							client.work = false
							delete(h.clients, client)
							close(client.send)
							_ = client.conn.Close()
							break
						}

					}
				}
			}
		}
	}

}
