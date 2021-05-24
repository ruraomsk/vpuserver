package mainScreen

import (
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ruraomsk/VPUserver/model/accToken"
	u "github.com/ruraomsk/VPUserver/utils"
	"net/http"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

//HMainScreen обработчик открытия сокета
func HMainScreen(c *gin.Context, hub *HubMainScreen) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		u.SendRespond(c, u.Message(http.StatusBadRequest, "bad socket connect"))
		return
	}

	accInfo := new(accToken.Token)
	tokenInfo := new(jwt.Token)

	cookie, err := c.Cookie("Authorization")
	//Проверка куков получили ли их вообще
	if err != nil {
		cookie = ""
	}
	accInfo.IP = c.ClientIP()
	client := &ClientMS{hub: hub, conn: conn, send: make(chan mSResponse, 256), cInfo: accInfo, rawToken: tokenInfo.Raw, cookie: cookie}
	client.hub.register <- client

	go client.writePump()
	go client.readPump()
}
