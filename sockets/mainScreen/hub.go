package mainScreen

import (
	"github.com/ruraomsk/VPUserver/model/accToken"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/VPUserver/model/license"
	"reflect"
	"sync"
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

var mutex sync.Mutex

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
				mutex.Lock()
				if len(h.clients) > 0 {
					newPhones := getAllPhones()
					for client := range h.clients {
						if !client.isLogin {
							continue
						}
						if len(newPhones) != len(client.listPhone) {
							resp := newMainMess(typeRepaint, nil)
							resp.Data["phones"] = newPhones
							client.send <- resp
						} else {
							found := false
							for _, phn := range newPhones {
								pho, is := client.listPhone[phn.Login]
								if !is {
									found = true
									break
								}
								if !reflect.DeepEqual(&phn, &pho) {
									found = true
									break
								}
							}
							if !found {
								for _, pho := range client.listPhone {
									phn, is := newPhones[pho.Login]
									if !is {
										found = true
										break
									}
									if !reflect.DeepEqual(&phn, &pho) {
										found = true
										break
									}
								}
							}
							if found {
								resp := newMainMess(typeRepaint, nil)
								resp.Data["phones"] = newPhones
								client.send <- resp
							}

						}
						client.listPhone = newPhones
					}

				}
				mutex.Unlock()
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
				mutex.Lock()
				h.clients[client] = true
				mutex.Unlock()
			}
		case client := <-h.unregister:
			{
				if _, ok := h.clients[client]; ok {
					mutex.Lock()
					delete(h.clients, client)
					close(client.send)
					mutex.Lock()
					_ = client.conn.Close()
				}
			}
		case mess := <-h.broadcast:
			{
				for client := range h.clients {
					select {
					case client.send <- mess:
					default:
						mutex.Lock()
						delete(h.clients, client)
						close(client.send)
						mutex.Unlock()
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
		}
	}

}
