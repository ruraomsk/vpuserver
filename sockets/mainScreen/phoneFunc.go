package mainScreen

import (
	"encoding/json"
	"github.com/ruraomsk/VPUserver/model/data"
	"github.com/ruraomsk/device/dataBase"
	"reflect"
	"sort"
	"strings"
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
func mapToArray(phones map[string]dataBase.Phone) []dataBase.Phone {
	res := make([]dataBase.Phone, 0)
	for _, ph := range phones {
		res = append(res, ph)
	}
	sort.Slice(res, func(i, j int) bool {
		if strings.Compare(res[i].Login, res[j].Login) > 0 {
			return true
		}
		return false
	})
	return res
}
func getAreas() map[int]string {
	res := make(map[int]string)
	db, id := data.GetDB()
	defer data.FreeDB(id)
	rows, err := db.Query("select area,namearea from area;")
	if err != nil {
		return res
	}
	var area int
	var namearea string
	for rows.Next() {
		_ = rows.Scan(&area, &namearea)
		res[area] = namearea
	}
	return res
}
func (h *HubMainScreen) sendFhones() {
	if len(h.clients) > 0 {
		newPhones := getAllPhones()
		for client := range h.clients {
			if !client.isLogin || !client.work {
				client.listPhone = make(map[string]dataBase.Phone)
				continue
			}
			if len(newPhones) != len(client.listPhone) {
				resp := newMainMess(typePhoneTable, nil)
				resp.Data["phones"] = mapToArray(newPhones)
				resp.Data["areas"] = getAreas()
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
					resp := newMainMess(typePhoneTable, nil)
					resp.Data["phones"] = mapToArray(newPhones)
					resp.Data["areas"] = getAreas()
					client.send <- resp
				}

			}
			client.listPhone = newPhones
		}

	}

}
