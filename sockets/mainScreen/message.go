package mainScreen

var (
	typeJump           = "jump"
	typeMapInfo        = "mapInfo"
	typeTFlight        = "tflight"
	typeRepaint        = "repaint"
	typePhoneTable     = "phoneTable"
	typeEditCrossUsers = "editCrossUsers"
	typeLogin          = "login"
	typeLogOut         = "logOut"
	typeChangeAccount  = "changeAcc"
	typeError          = "error"
	typeClose          = "close"
	typeCheckConn      = "checkConn"
	typeDButton        = "dispatch"

	errParseType = "Сервер не смог обработать запрос"
)

//MapSokResponse структура для отправки сообщений (map)
type mSResponse struct {
	Type string                 `json:"type"` //тип сообщения
	Data map[string]interface{} `json:"data"` //данные
}

//newMapMess создание нового сообщения
func newMainMess(mType string, data map[string]interface{}) mSResponse {
	var resp mSResponse
	resp.Type = mType
	if data != nil {
		resp.Data = data
	} else {
		resp.Data = make(map[string]interface{})
	}
	return resp
}

//ErrorMessage структура ошибки
type ErrorMessage struct {
	Error string `json:"error"`
}
