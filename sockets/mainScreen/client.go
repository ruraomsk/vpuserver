package mainScreen

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"github.com/ruraomsk/TLServer/logger"
	"github.com/ruraomsk/VPUserver/model/accToken"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/VPUserver/sockets"
	"github.com/ruraomsk/device/dataBase"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	phoneTick           = time.Second * 5
	checkTokensValidity = time.Minute * 1
)

var UserLogoutGS chan string //канал для закрытия сокетов, пользователя который вышел из системы

//ClientMS информация о подключившемся пользователе
type ClientMS struct {
	hub  *HubMainScreen
	conn *websocket.Conn
	send chan mSResponse

	cInfo     *accToken.Token
	rawToken  string
	cookie    string
	isLogin   bool
	listPhone map[string]dataBase.Phone
	work      bool
}

//readPump обработчик чтения сокета
func (c *ClientMS) readPump() {
	//если нужно указать лимит пакета
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { _ = c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, p, err := c.conn.ReadMessage()
		if err != nil {
			c.hub.unregister <- c
			c.work = false
			return
		}
		//ну отправка и отправка
		typeSelect, err := sockets.ChoseTypeMessage(p)
		if err != nil {
			logger.Error.Printf("|IP: %v |Login: %v |Resource: /mainScreen |Message: %v \n", c.cInfo.IP, c.cInfo.Login, err.Error())
			resp := newMainMess(typeError, nil)
			resp.Data["message"] = ErrorMessage{Error: errParseType}
			c.send <- resp
			continue
		}
		switch typeSelect {
		case typePhoneTable: //отправка default
			{
				c.listPhone = make(map[string]dataBase.Phone)
			}
		case typeLogin: //отправка default
			{
				var (
					account  = &data.Account{}
					token    *accToken.Token
					tokenStr string
				)
				_ = json.Unmarshal(p, &account)
				resp := newMainMess(typeLogin, nil)
				var status bool
				resp.Data, token, tokenStr, status = logIn(account.Login, account.Password, c.conn.RemoteAddr().String())
				if token != nil {
					//делаем выход из аккаунта
					for client := range c.hub.clients {
						if client.cInfo.Login == account.Login {
							//logOutSockets(account.Login)
							respLO := newMainMess(typeLogOut, nil)
							client.send <- respLO
							break
						}
					}
					c.cInfo = token
					c.cookie = tokenStr
				}
				c.send <- resp
				c.isLogin = status
			}
		case typeChangeAccount:
			{
				var (
					account  = &data.Account{}
					token    *accToken.Token
					tokenStr string
				)
				var status bool
				_ = json.Unmarshal(p, &account)
				resp := newMainMess(typeLogin, nil)
				resp.Data, token, tokenStr, status = logIn(account.Login, account.Password, c.conn.RemoteAddr().String())
				if token != nil {
					//делаем выход из аккаунта
					respLO := newMainMess(typeLogOut, nil)
					status := logOut(c.cInfo.Login)
					if status {
						logOutSockets(c.cInfo.Login)
						c.cInfo = token
						c.cookie = tokenStr
					}
					c.send <- respLO
				}
				c.send <- resp
				c.isLogin = status
			}
		case typeLogOut: //отправка default
			{
				if c.cInfo.Login != "" {
					resp := newMainMess(typeLogOut, nil)
					status := logOut(c.cInfo.Login)
					if status {
						resp.Data["authorizedFlag"] = false
						//logOutSockets(c.cInfo.Login)
					}
					c.cInfo = new(accToken.Token)
					c.cookie = ""
					c.send <- resp
					c.isLogin = false
				}
			}
		case typeCheckConn: //отправка default
			{
				//resp := newMainMess(typeCheckConn, nil)
				//statusDB := false
				//db, id := data.GetDB()
				//if db != nil {
				//	statusDB = true
				//	data.FreeDB(id)
				//}
				//resp.Data["statusBD"] = statusDB
				//var tcpPackage = tcpConnect.TCPMessage{
				//	TCPType:     tcpConnect.TypeState,
				//	User:        c.cInfo.Login,
				//	Idevice:     -1,
				//	Data:        0,
				//	From:        tcpConnect.FromMapSoc,
				//	CommandType: typeDButton,
				//}
				//tcpPackage.SendToTCPServer()
				//
				//c.send <- resp
			}
		default:
			{
				resp := newMainMess("type", nil)
				resp.Data["type"] = typeSelect
				c.send <- resp
			}
		}
	}
}

//writePump обработчик записи в сокет
func (c *ClientMS) writePump() {
	pingTick := time.NewTicker(pingPeriod)
	workTick := time.NewTicker(50 * time.Millisecond)
	defer func() {
		pingTick.Stop()
		workTick.Stop()
	}()
	for {
		select {
		case mess, ok := <-c.send:
			{
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if !ok {
					_ = c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, "канал был закрыт"))
					return
				}
				_ = c.conn.WriteJSON(mess)
				// Add queued chat messages to the current websocket message.
				n := len(c.send)
				for i := 0; i < n; i++ {
					_ = c.conn.WriteJSON(<-c.send)
				}
			}
		case <-pingTick.C:
			{
				_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
				if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
					return
				}
			}
		case <-workTick.C:
			{
				if !c.work {
					return
				}
			}
		}
	}
}
