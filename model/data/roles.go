package data

import (
	"encoding/json"
	"fmt"
	"github.com/ruraomsk/VPUserver/model/accToken"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	u "github.com/ruraomsk/VPUserver/utils"
)

//RoleInfo глабальная переменная для обращения к мапам
var RoleInfo RoleData

//RoleData структура, включающая всю информацию о ролях, привелегиях, и маршрутах
type RoleData struct {
	Mux          sync.Mutex
	MapRoles     map[string][]int     //роли
	MapPermisson map[int]Permission   //привилегии
	MapRoutes    map[string]RouteInfo //маршруты
}

//RoleAccess информация наборах ролей и полномочий
type RoleAccess struct {
	Roles      []Role       `json:"roles"`       //массив ролей
	Permission []Permission `json:"permissions"` //массив разрешений
	Routes     []RouteInfo  `json:"routes"`      //массив маршрутов
}

//Role информация о роли
type Role struct {
	Name string `json:"name"`        //название роли
	Perm []int  `json:"permissions"` //массив полномочий
}

//Privilege структура  для запросов к БД
type Privilege struct {
	Role         Role     `json:"role"`   //информация о роли пользователя
	Region       string   `json:"region"` //регион пользователя
	Area         []string `json:"area"`   //массив районов пользователя
	PrivilegeStr string   `json:"-"`      //строка для декодирования
}

//Permission структура полномочий содержит ID, команду и описание команды
type Permission struct {
	ID          int    `json:"id"`          //ID порядковый номер
	Visible     bool   `json:"visible"`     //флаг отображения пользователю
	Description string `json:"description"` //описание команды
}

//shortPermission структура полномойчий содержит ID, команду и описание команды урезанный вид для отправки пользователю
type shortPermission struct {
	ID          int    `json:"id"`          //ID порядковый номер
	Description string `json:"description"` //описание команды
}

//RouteInfo информация о всех расписанных маршрутах
type RouteInfo struct {
	ID          int    `json:"id"`          //уникальный номер маршрута
	Permission  int    `json:"permission"`  //номер разрешения к которому относится этот маршрут
	Path        string `json:"path"`        //путь (url) обращения к ресурсу
	Description string `json:"description"` //описание маршрута
}

//DisplayInfoForAdmin отображение информации о пользователях для администраторов
func (privilege *Privilege) DisplayInfoForAdmin(accInfo *accToken.Token) u.Response {
	var (
		sqlStr   string
		shortAcc []ShortAccount
	)
	err := privilege.ReadFromBD(accInfo.Login)
	if err != nil {
		return u.Message(http.StatusInternalServerError, "display info: Privilege error")
	}
	//если нужно из списка исключить пользователя раскомментировать строчки
	//sqlStr = fmt.Sprintf("select login, work_time, privilege, description from public.accounts where login != '%s'", mapContx["login"])
	sqlStr = fmt.Sprintf("select login, work_time, privilege, description from public.accounts ")
	if !strings.EqualFold(privilege.Region, "*") {
		//sqlStr += fmt.Sprintf(`and privilege::jsonb @> '{"region":"%s"}'::jsonb`, privilege.Region)
		sqlStr += fmt.Sprintf(`where privilege::jsonb @> '{"region":"%s"}'::jsonb`, privilege.Region)
	}
	db, id := GetDB()
	defer FreeDB(id)
	rowsTL, err := db.Query(sqlStr)
	if err != nil {
		return u.Message(http.StatusBadRequest, "display info: Bad request")
	}
	for rowsTL.Next() {
		var tempSA = ShortAccount{}
		err := rowsTL.Scan(&tempSA.Login, &tempSA.WorkTime, &tempSA.Privilege, &tempSA.Description)
		if err != nil {
			return u.Message(http.StatusBadRequest, "display info: Bad request")
		}
		var tempPrivilege = Privilege{}
		tempPrivilege.PrivilegeStr = tempSA.Privilege
		err = tempPrivilege.ConvertToJson()
		if err != nil {
			return u.Message(http.StatusInternalServerError, "display info: Privilege json error")
		}
		tempSA.Role.Name = tempPrivilege.Role.Name

		//выбираю привелегии которые не ключены в шаблон роли

		RoleInfo.Mux.Lock()
		for _, val1 := range tempPrivilege.Role.Perm {
			flag1, flag2 := false, false
			for _, val2 := range RoleInfo.MapRoles[tempSA.Role.Name] {
				if val2 == val1 {
					flag1 = true
					break
				}
			}
			for _, val3 := range tempSA.Role.Perm {
				if val3 == val1 {
					flag2 = true
					break
				}
			}
			if !flag1 && !flag2 {
				tempSA.Role.Perm = append(tempSA.Role.Perm, val1)
			}
		}
		RoleInfo.Mux.Unlock()

		if tempSA.Role.Perm == nil {
			tempSA.Role.Perm = make([]int, 0)
		}
		if tempSA.Login != AutomaticLogin {
			shortAcc = append(shortAcc, tempSA)
		}
	}

	resp := u.Message(http.StatusOK, "display information for Admins")

	//собираем в кучу роли
	RoleInfo.Mux.Lock()
	var roles []string

	if accInfo.Login == AutomaticLogin {
		roles = append(roles, "Admin")
	}

	for roleName := range RoleInfo.MapRoles {
		if (accInfo.Role == "Admin") && (roleName == "Admin") {
			continue
		}
		if (accInfo.Role == "RegAdmin") && ((roleName == "Admin") || (roleName == "RegAdmin")) {
			continue
		}
		roles = append(roles, roleName)
	}
	resp.Obj["roles"] = roles

	//собираю в кучу разрешения без указания команд
	chosenPermisson := make(map[int]shortPermission)
	for key, value := range RoleInfo.MapPermisson {
		for _, permCreator := range privilege.Role.Perm {
			if value.Visible && permCreator == value.ID {
				var shValue shortPermission
				shValue.transform(value)
				chosenPermisson[key] = shValue
			}
		}
	}
	resp.Obj["permInfo"] = chosenPermisson
	RoleInfo.Mux.Unlock()
	resp.Obj["accInfo"] = shortAcc
	return resp
}

//transform преобразование из расшириных разрешений к коротким
func (shPerm *shortPermission) transform(perm Permission) {
	shPerm.Description = perm.Description
	shPerm.ID = perm.ID
}

//ReadRoleAccessFile чтение RoleAccess файла
func (roleAccess *RoleAccess) ReadRoleAccessFile() (err error) {
	file, err := ioutil.ReadFile(`./configs/RoleAccess.json`)
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, roleAccess)
	if err != nil {
		return err
	}
	return err
}

//ToSqlStrUpdate запись привилегий в базу
func (privilege *Privilege) WriteRoleInBD(login string) (err error) {
	privilegeStr, _ := json.Marshal(privilege)
	db, id := GetDB()
	defer FreeDB(id)

	_, err = db.Exec(`UPDATE public.accounts set privilege = $1 where login = $2`, string(privilegeStr), login)
	return
}

//ReadFromBD прочитать данные из бд и разобрать
func (privilege *Privilege) ReadFromBD(login string) error {
	var privilegeStr string
	db, id := GetDB()
	defer FreeDB(id)
	err := db.QueryRow(`SELECT privilege FROM public.accounts WHERE login = $1`, login).Scan(&privilegeStr)
	if err != nil {
		return err
	}
	err = json.Unmarshal([]byte(privilegeStr), privilege)
	if err != nil {
		return err
	}
	return nil
}

//ConvertToJson из строки в структуру
func (privilege *Privilege) ConvertToJson() (err error) {
	err = json.Unmarshal([]byte(privilege.PrivilegeStr), privilege)
	if err != nil {
		return err
	}
	return nil
}

//NewPrivilege созданеие привелегии
func NewPrivilege(role, region string, area []string) *Privilege {
	var privilege Privilege
	RoleInfo.Mux.Lock()
	if _, ok := RoleInfo.MapRoles[role]; ok {
		privilege.Role.Name = role
	} else {
		privilege.Role.Name = "Viewer"
	}

	for _, permission := range RoleInfo.MapRoles[privilege.Role.Name] {
		privilege.Role.Perm = append(privilege.Role.Perm, permission)
	}
	RoleInfo.Mux.Unlock()
	if region == "" {
		privilege.Region = "0"
	} else {
		privilege.Region = region
	}

	if len(region) == 0 {
		privilege.Area = []string{"0"}
	} else {
		privilege.Area = area
	}

	return &privilege
}

//AccessCheck проверка разрешения на доступ к ресурсу
func AccessCheck(login string, acts ...int) map[int]bool {
	accessMap := make(map[int]bool)
	for _, act := range acts {
		accessMap[act] = false
	}
	privilege := Privilege{}
	//Проверил соответствует ли роль которую мне дали с ролью установленной в БД
	err := privilege.ReadFromBD(login)
	if err != nil {

		return accessMap
	}
	for _, act := range acts {
		for _, perm := range privilege.Role.Perm {
			if perm == act {
				accessMap[act] = true
				break
			}
		}
	}
	accessMap[10] = true
	return accessMap
}
