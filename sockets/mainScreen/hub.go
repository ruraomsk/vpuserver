package mainScreen

import (
	"github.com/ruraomsk/VPUserver/model/accToken"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/VPUserver/model/license"
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
	crossReadTick := time.NewTicker(crossTick)
	defer func() {
		crossReadTick.Stop()
		checkValidityTicker.Stop()
	}()

	for {
		select {
		case <-crossReadTick.C:
			{
				if len(h.clients) > 0 {
					//if len(newTFs) != len(oldTFs) {
					//	resp := newMapMess(typeRepaint, nil)
					//	resp.Data["tflight"] = newTFs
					//	data.FillMapAreaZone()
					//	data.CacheArea.Mux.Lock()
					//	resp.Data["areaZone"] = data.CacheArea.Areas
					//	data.CacheArea.Mux.Unlock()
					//	for client := range h.clients {
					//		client.send <- resp
					//	}
					//} else {
					//	var (
					//		tempTF   []data.TrafficLights
					//		flagFill = false
					//	)
					//	for _, nTF := range newTFs {
					//		var flagAdd = true
					//		for _, oTF := range oldTFs {
					//			if oTF.Idevice == nTF.Idevice {
					//				flagAdd = false
					//				if oTF.Sost.Num != nTF.Sost.Num || oTF.Description != nTF.Description || oTF.Points != nTF.Points {
					//					flagAdd = true
					//				}
					//				if oTF.Subarea != nTF.Subarea {
					//					flagAdd = true
					//					flagFill = true
					//				}
					//				break
					//			}
					//		}
					//		if flagAdd {
					//			tempTF = append(tempTF, nTF)
					//		}
					//	}
					//	if len(tempTF) > 0 {
					//		resp := newMapMess(typeTFlight, nil)
					//		if flagFill {
					//			data.FillMapAreaZone()
					//			data.CacheArea.Mux.Lock()
					//			resp.Data["areaZone"] = data.CacheArea.Areas
					//			data.CacheArea.Mux.Unlock()
					//		}
					//		resp.Data["tflight"] = tempTF
					//		for client := range h.clients {
					//			client.send <- resp
					//		}
					//	}
					//}
					//oldTFs = newTFs
				}
			}
		case client := <-h.register:
			{
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
					}
					client.send <- resp
				}

				h.clients[client] = true
			}
		case client := <-h.unregister:
			{
				if _, ok := h.clients[client]; ok {
					delete(h.clients, client)
					close(client.send)
					_ = client.conn.Close()
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
		}
	}

}
