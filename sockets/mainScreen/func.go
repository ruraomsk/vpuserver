package mainScreen

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/ruraomsk/VPUserver/model/accToken"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/VPUserver/model/license"
	"golang.org/x/crypto/bcrypt"
	"strings"
	"time"
)

//checkToken проверка токена для вебсокета
func checkToken(cookie string, ip string) (flag bool, t *accToken.Token) {
	if cookie == "" {
		return false, nil
	}
	//токен приходит строкой в формате {слово пробел слово} разделяем строку и забираем нужную нам часть
	splitted := strings.Split(cookie, " ")
	if len(splitted) != 2 {
		return false, nil
	}

	//берем часть где хранится токен
	tokenSTR := splitted[1]
	tk := &accToken.Token{}

	token, err := jwt.ParseWithClaims(tokenSTR, tk, func(token *jwt.Token) (interface{}, error) {
		return []byte(license.LicenseFields.TokenPass), nil
	})

	//не правильный токен возвращаем ошибку с кодом 403
	if err != nil {
		return false, nil
	}

	//Проверка на уникальность токена
	var (
		userPrivilege  data.Privilege
		tokenStrFromBd string
	)
	db, id := data.GetDB()
	defer data.FreeDB(id)
	rows, err := db.Query(`SELECT token, privilege FROM public.accounts WHERE login = $1`, tk.Login)
	if err != nil {
		return false, nil
	}
	for rows.Next() {
		_ = rows.Scan(&tokenStrFromBd, &userPrivilege.PrivilegeStr)
	}

	if tokenSTR != tokenStrFromBd || tk.IP != ip || !token.Valid {
		return false, nil
	}

	//проверка токен пришел от правильного URL

	//проверка правильности роли для указанного пользователя
	_ = userPrivilege.ConvertToJson()
	if userPrivilege.Role.Name != tk.Role {
		return false, nil
	}

	return true, tk
}

//logIn обработчик авторизации пользователя в системе
func logIn(login, password, ip string) (map[string]interface{}, *accToken.Token, string, bool) {
	db, id := data.GetDB()
	defer data.FreeDB(id)
	resp := make(map[string]interface{})
	ipSplit := strings.Split(ip, ":")
	account := &data.Account{}
	//Забираю из базы запись с подходящей почтой
	rows, err := db.Query(`SELECT login, password, work_time, description FROM public.accounts WHERE login=$1`, login)
	if rows == nil {
		resp["status"] = false
		resp["message"] = fmt.Sprintf("Неверно указан логин или пароль")
		return resp, nil, "", false
	}
	if err != nil {
		resp["status"] = false
		resp["message"] = "Потеряно соединение с сервером БД"
		return resp, nil, "", false
	}
	for rows.Next() {
		_ = rows.Scan(&account.Login, &account.Password, &account.WorkTime, &account.Description)
	}

	//Авторизировались добираем полномочия
	privilege := data.Privilege{}
	err = privilege.ReadFromBD(account.Login)
	if err != nil {
		resp["status"] = false
		resp["message"] = fmt.Sprintf("Неверно указан логин или пароль")
		return resp, nil, "", false
	}

	//Сравниваю хэши полученного пароля и пароля взятого из БД
	err = bcrypt.CompareHashAndPassword([]byte(account.Password), []byte(password))
	if err != nil && err == bcrypt.ErrMismatchedHashAndPassword {
		resp["status"] = false
		resp["message"] = fmt.Sprintf("Неверно указан логин или пароль")
		return resp, nil, "", false
	}
	//Залогинились, создаем токен
	account.Password = ""
	tk := &accToken.Token{
		Login:       account.Login,
		IP:          ipSplit[0],
		Role:        privilege.Role.Name,
		Region:      privilege.Region,
		Area:        privilege.Area,
		Permission:  privilege.Role.Perm,
		Description: account.Description,
	}
	//врямя выдачи токена
	tk.IssuedAt = time.Now().Unix()
	//время когда закончится действие токена
	tk.ExpiresAt = time.Now().Add(time.Minute * account.WorkTime).Unix()

	token := jwt.NewWithClaims(jwt.GetSigningMethod("HS256"), tk)
	tokenStr, _ := token.SignedString([]byte(license.LicenseFields.TokenPass))
	account.Token = tokenStr
	//сохраняем токен в БД чтобы точно знать что дейтвителен только 1 токен

	_, err = db.Exec(`UPDATE public.accounts SET token = $1 WHERE login = $2`, account.Token, account.Login)
	if err != nil {
		resp["status"] = false
		resp["message"] = "Потеряно соединение с сервером БД"
		return resp, nil, "", false
	}

	//Формируем ответ
	resp["status"] = true
	resp["login"] = account.Login
	resp["token"] = tokenStr
	resp["role"] = privilege.Role.Name
	resp["access"] = data.AccessCheck(login, 2, 5, 6, 7, 8, 9, 10)
	resp["authorizedFlag"] = true
	resp["description"] = account.Description
	resp["region"] = privilege.Region
	//собрать в районы с их названиями
	var areaMap = make(map[string]string)
	resp["area"] = areaMap

	return resp, tk, tokenStr, true
}

//logOut выход из учетной записи
func logOut(login string) bool {
	db, id := data.GetDB()
	defer data.FreeDB(id)

	_, err := db.Exec("UPDATE public.accounts SET token = $1 where login = $2", "", login)
	if err != nil {
		return false
	}
	return true
}

//logOutSockets закрытие всех сокетов по действию logout на основном экране
func logOutSockets(login string) {

	UserLogoutGS <- login
}
