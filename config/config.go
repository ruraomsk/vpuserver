package config

import (
	"fmt"
	"github.com/ruraomsk/VPUserver/tcpConnect"
	//"github.com/ruraomsk/TLServer/internal/app/tcpConnect"
)

//GlobalConfig глобальная переменная для структуры конфиг
var GlobalConfig *Config

//DBConfig структура с полями для ДБ
type DBConfig struct {
	Name            string `toml:"db_name"`            //имя БД
	Password        string `toml:"db_password"`        //пароль доступа к БД
	User            string `toml:"db_user"`            //пользователя для обращения к бд
	Type            string `toml:"db_type"`            //тип бд
	Host            string `toml:"db_host"`            //ip сервера бд
	Port            string `toml:"db_port"`            //порт для обращения к бд
	SetMaxOpenConst int    `toml:"db_SetMaxOpenConst"` //максимальное количество пустых соединений с бд
	SetMaxIdleConst int    `toml:"db_SetMaxIdleConst"` //максимальное количество соединенияй с бд
}

//Config структура с объявлением всех переменных config.toml файла
type Config struct {
	TCPConfig  tcpConnect.TCPConfig `toml:"tcpServer"`   //информация о tcp соединении с сервером устройств
	DBConfig   DBConfig             `toml:"database"`    //информация о соединении с базой данных
	StaticPath string               `toml:"static_path"` //путь до ресурсов кроса
	Region     int                  `toml:"region"`      //Регион в котором работает сервер
}

//GetDBurl сформировать url для подключения к БД
func (dbConfig *DBConfig) GetDBurl() string {
	return fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable password=%s", dbConfig.Host, dbConfig.User, dbConfig.Name, dbConfig.Password)
}

//NewConfig создание конфига
func NewConfig() *Config {
	return &Config{}
}
