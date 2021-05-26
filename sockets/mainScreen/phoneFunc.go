package mainScreen

import (
	"encoding/json"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/device/dataBase"
)

func getAllPhones() map[string]dataBase.Phone {
	phones := make(map[string]dataBase.Phone)
	db, id := data.GetDB()
	defer data.FreeDB(id)
	rows, err := db.Query("select phone from phones;")
	if err != nil {
		return phones
	}
	var phone dataBase.Phone
	var buff []byte
	for rows.Next() {
		_ = rows.Scan(&buff)
		_ = json.Unmarshal(buff, &phone)
		phones[phone.Login] = phone
	}
	return phones

}
