package data

import (
	_ "database/sql"
	"fmt"
	"github.com/ruraomsk/VPUserver/config"
	"sync"

	_ "github.com/jackc/pgx"
	"github.com/jmoiron/sqlx"
	"github.com/ruraomsk/TLServer/logger"
)

var (
	accountsTable = `
	CREATE TABLE if not exists accounts (
		description text,
		login text PRIMARY KEY,
		password text,
		work_time bigint,
		token text,
		privilege jsonb
	)
	WITH (
		autovacuum_enabled = true		
	);`

	//FirstCreate флаг первого создания базы
	FirstCreate bool
)

type usedDb struct {
	db   *sqlx.DB
	used bool
}

var dbPool []usedDb
var mutex sync.Mutex
var first = true

//ConnectDB подключение к БД
func ConnectDB() error {
	if first {
		dbPool = make([]usedDb, 0)
		first = false
		for i := 0; i < config.GlobalConfig.DBConfig.SetMaxOpenConst; i++ {
			conn, err := sqlx.Open(config.GlobalConfig.DBConfig.Type, config.GlobalConfig.DBConfig.GetDBurl())
			if err != nil {
				return err
			}
			dbPool = append(dbPool, usedDb{db: conn, used: false})
		}
	}
	db, id := GetDB()
	_, err := db.Exec(`SELECT * FROM public."accounts";`)
	if err != nil {
		fmt.Println("accounts table not found - created")
		logger.Info.Println("|Message: accounts table not found - created")
		db.MustExec(accountsTable)
		FirstCreate = true
	}

	FreeDB(id)
	return nil
}

//GetDB обращение к БД
func GetDB() (db *sqlx.DB, id int) {
	mutex.Lock()
	defer mutex.Unlock()
	for i, d := range dbPool {
		if !d.used {
			dbPool[i].used = true
			return d.db, i
		}
	}
	logger.Error.Printf("dbase закончился пул соединений")
	return nil, 0
}
func FreeDB(id int) {
	mutex.Lock()
	defer mutex.Unlock()
	if id < 0 || id >= len(dbPool) {
		logger.Error.Printf("dbase freeDb неверный индекс %d", id)
		return
	}
	dbPool[id].used = false
}
