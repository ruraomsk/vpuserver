package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	validation "github.com/go-ozzo/ozzo-validation"
	"github.com/ruraomsk/VPUserver/model/license"
	u "github.com/ruraomsk/VPUserver/utils"
	"golang.org/x/crypto/bcrypt"
)

//Account структура аккаунта пользователя
type Account struct {
	Description string        `json:"description"` //описание арм
	Login       string        `json:"login"`       //Имя пользователя
	Password    string        `json:"password"`    //Пароль
	WorkTime    time.Duration `json:"workTime"`    //Время работы пользователя в часах
	YaMapKey    string        `json:"ya_key"`      //Ключ доступа к яндекс карте
	Token       string        `json:"token"`       //Токен пользователя
}

var (
	AutomaticLogin     = "TechAutomatic"            //Пользователь для суперпользователя :D
	errorConnectDB     = "соединение с БД потеряно" //стандартная ошибка
	errorDuplicateUser = "такой пользователь уже существует"
	passLong           = 10
	AccAction          chan string
)

//Validate проверка аккаунда в бд
func (data *Account) Validate() error {
	err := validation.ValidateStruct(data,
		validation.Field(&data.Login, validation.Required, validation.Length(4, 100)),
		validation.Field(&data.Description, validation.Required, validation.Length(1, 255)),
	)
	if data.Login == "Global" {
		return errors.New("этот логин не может быть создан")
	}
	if err != nil {
		return err
	}
	//логин аккаунта должен быть уникальным
	temp := &Account{}
	db, id := GetDB()
	defer FreeDB(id)
	rows, err := db.Query(`SELECT id, login, password FROM public.accounts WHERE login=$1`, data.Login)
	if rows != nil && err != nil {
		return errors.New("connection error, please try again")
	}
	if temp.Login != "" {
		return errors.New("login already in use by another user")
	}
	return nil
}

//Create создание аккаунта для пользователей
func (data *Account) Create(privilege Privilege) u.Response {
	var (
		count int
		login string
	)
	db, id := GetDB()
	defer FreeDB(id)
	if err := db.QueryRow(`SELECT count(*) FROM public.accounts`).Scan(&count); err != nil {
		return u.Message(http.StatusInternalServerError, errorConnectDB)
	}
	if (count - 1) >= license.LicenseFields.NumAcc {
		return u.Message(http.StatusOK, "ограничение по количеству аккаунтов")
	}
	if err := data.Validate(); err != nil {
		return u.Message(http.StatusBadRequest, err.Error())
	}
	pass := u.GenerateRandomKey(passLong)
	data.Password = pass
	//Отдаем ключ для yandex map
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	data.Password = string(hashedPassword)
	row := db.QueryRow(`INSERT INTO  public.accounts (login, password, work_time, description) VALUES ($1, $2, $3, $4) RETURNING login`,
		data.Login, data.Password, data.WorkTime, data.Description)
	if err := row.Scan(&login); err != nil {
		return u.Message(http.StatusInternalServerError, errorDuplicateUser)
	}
	if data.Login != login {
		return u.Message(http.StatusBadRequest, "ошибка создания пользователя")
	}
	RoleInfo.Mux.Lock()
	privilege.Role.Perm = append(privilege.Role.Perm, RoleInfo.MapRoles[privilege.Role.Name]...)
	RoleInfo.Mux.Unlock()
	if err := privilege.WriteRoleInBD(data.Login); err != nil {
		return u.Message(http.StatusBadRequest, errorConnectDB)
	}
	resp := u.Message(http.StatusOK, "аккаунт создан")
	resp.Obj["pass"] = pass
	resp.Obj["login"] = data.Login
	return resp
}

//Update обновление данных аккаунта
func (data *Account) Update(privilege Privilege) u.Response {
	RoleInfo.Mux.Lock()
	privilege.Role.Perm = append(privilege.Role.Perm, RoleInfo.MapRoles[privilege.Role.Name]...)
	RoleInfo.Mux.Unlock()
	privStr, _ := json.Marshal(privilege)
	db, id := GetDB()
	defer FreeDB(id)

	_, err := db.Exec(`UPDATE public.accounts SET privilege = $1, work_time = $2, description = $3, token = $4 WHERE login = $5`, string(privStr), data.WorkTime, data.Description, "", data.Login)
	if err != nil {
		resp := u.Message(http.StatusInternalServerError, fmt.Sprintf("Account update error: %s", err.Error()))
		return resp
	}
	resp := u.Message(http.StatusOK, "аккаунт обновлен")
	AccAction <- data.Login
	return resp
}

//Delete удаление аккаунта из БД
func (data *Account) Delete() u.Response {
	db, id := GetDB()
	defer FreeDB(id)
	_, err := db.Exec(`DELETE FROM public.accounts WHERE login = $1`, data.Login)
	if err != nil {
		resp := u.Message(http.StatusInternalServerError, "data deletion error "+err.Error())
		return resp
	}
	resp := u.Message(http.StatusOK, "аккаунт удален")
	AccAction <- data.Login
	return resp
}

//ResetPass сброс пароля
func (data *Account) ResetPass() u.Response {
	pass := u.GenerateRandomKey(passLong)
	data.Password = pass
	//Отдаем ключ для yandex map
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	data.Password = string(hashedPassword)
	db, id := GetDB()
	defer FreeDB(id)

	_, err := db.Exec(`UPDATE public.accounts SET password = $1, token = $2 WHERE login = $3`, data.Password, "", data.Login)
	if err != nil {
		resp := u.Message(http.StatusInternalServerError, fmt.Sprintf("Ошибка сброса пароля: %s", err.Error()))
		return resp
	}
	resp := u.Message(http.StatusOK, "Пароль изменен")
	AccAction <- data.Login
	resp.Obj["pass"] = pass
	resp.Obj["login"] = data.Login
	return resp
}

//ChangePW изменение пароля пользователя
func (data *Account) ChangePW() u.Response {
	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(data.Password), bcrypt.DefaultCost)
	data.Password = string(hashedPassword)

	sqlStr := fmt.Sprintf(`UPDATE public.accounts SET password = '%v' WHERE login = '%v'; UPDATE public.accounts SET token = '' WHERE login = '%v'`, data.Password, data.Login, data.Login)
	db, id := GetDB()
	defer FreeDB(id)
	_, err := db.Exec(sqlStr)
	if err != nil {
		resp := u.Message(http.StatusInternalServerError, "password change error "+err.Error())
		return resp
	}
	resp := u.Message(http.StatusOK, "password changed")
	AccAction <- data.Login
	return resp
}

//ParserBoxPointsUser заполняет BoxPoint
func (data *Account) ParserBoxPointsUser() (err error) {
	var (
		privilege = Privilege{}
	)
	err = privilege.ReadFromBD(data.Login)
	if err != nil {
		return fmt.Errorf("parserPoints. Privilege error: %s", err.Error())
	}
	var sqlString = `SELECT Min(dgis[0]) as "Y0", Min(convTo360(dgis[1])) as "X0", Max(dgis[0]) as "Y1", Max(convTo360(dgis[1])) as "X1"  FROM public."cross"`
	if !strings.EqualFold(privilege.Region, "*") {
		sqlString = sqlString + fmt.Sprintf(" where region = %s;", privilege.Region)
	}
	return nil
}

//SuperCreate создание суперпользователя
func SuperCreate() {
	account := &Account{}
	account.Login = AutomaticLogin
	//Отдаем ключ для yandex map
	account.WorkTime = 24 * 60
	account.Password = "$2a$10$ZCWyIEfEVF3KGj6OUtIeSOQ3WexMjuAZ43VSO6T.QqOndn4HN1J6C"
	account.Description = "Tech"
	privilege := NewPrivilege("Admin", "*", []string{"*"})
	db, id := GetDB()
	defer FreeDB(id)

	_, _ = db.Exec(`INSERT INTO  public.accounts (login, password, work_time, description) VALUES ($1, $2, $3, $4)`,
		account.Login, account.Password, account.WorkTime, account.Description)
	////Записываю координаты в базу!!!
	_ = privilege.WriteRoleInBD(account.Login)

	//!!!!! НЕ забудь удалить все что вышел
	fmt.Println("Super created!")
}
