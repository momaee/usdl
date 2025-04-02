package sql

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/ardanlabs/usdl/chat/api/frontends/client/app"
	"github.com/ethereum/go-ethereum/common"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	dbDirName  = "db"
	dbFileName = "data.db"
)

type DB struct {
	db *gorm.DB
}

type myAccount struct {
	Singleton bool   `gorm:"primaryKey;default:true"`
	ID        string `gorm:"column:id"`
	Name      string `gorm:"column:name"`
}

type user struct {
	ID           string    `gorm:"primaryKey;column:id"`
	Name         string    `gorm:"column:name"`
	AppLastNonce uint64    `gorm:"column:app_last_nonce"`
	LastNonce    uint64    `gorm:"column:last_nonce"`
	Key          string    `gorm:"column:key"`
	Messages     []message `gorm:"foreignKey:UserID;column:messages"`
}

type message struct {
	ID     uint64 `gorm:"primaryKey;column:id"`
	Msg    string `gorm:"column:msg"`
	UserID string `gorm:"column:user_id"`
}

func NewDB(filePath string, myAccountID common.Address) (*DB, error) {
	dbFileDir := filepath.Join(filePath, dbDirName)
	os.MkdirAll(dbFileDir, os.ModePerm)

	fileName := filepath.Join(dbFileDir, dbFileName)
	db, err := gorm.Open(sqlite.Open(fileName), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("gorm open: %w", err)
	}

	if err := db.AutoMigrate(&user{}, &message{}, myAccount{}); err != nil {
		return nil, fmt.Errorf("auto migrate: %w", err)
	}

	if err := saveMyAccount(db, myAccountID); err != nil {
		return nil, fmt.Errorf("save my account: %w", err)
	}

	return &DB{db: db}, nil
}

func saveMyAccount(db *gorm.DB, myAccountID common.Address) error {
	var myAcc myAccount
	db.First(&myAcc)

	if myAcc.ID == "" {
		res := db.Save(&myAccount{
			Singleton: true,
			ID:        myAccountID.Hex(),
			Name:      "Anonymous",
		})
		if res.Error != nil {
			return fmt.Errorf("create my account: %w", res.Error)
		}
	}

	return nil
}

func (db *DB) MyAccount() app.MyAccount {
	var myAccount myAccount
	db.db.First(&myAccount)
	return app.MyAccount{
		ID:   common.HexToAddress(myAccount.ID),
		Name: myAccount.Name,
	}
}

func (db *DB) InsertContact(id common.Address, name string) (app.User, error) {
	res := db.db.Create(&user{
		ID:   id.Hex(),
		Name: name,
	})

	if res.Error != nil {
		return app.User{}, fmt.Errorf("insert contact: %w", res.Error)
	}

	return app.User{
		ID:   id,
		Name: name,
	}, nil
}

func (db *DB) QueryContactByID(id common.Address) (app.User, error) {
	var user user
	if err := db.db.Preload("Messages").Where("LOWER(id) = LOWER(?)", id.Hex()).First(&user).Error; err != nil {
		return app.User{}, fmt.Errorf("query contact: %s %w", id.Hex(), err)
	}

	msgs := make([]string, len(user.Messages))
	for i, msg := range user.Messages {
		msgs[i] = msg.Msg
	}

	return app.User{
		ID:           common.HexToAddress(user.ID),
		Name:         user.Name,
		AppLastNonce: user.AppLastNonce,
		LastNonce:    user.LastNonce,
		Key:          user.Key,
		Messages:     msgs,
	}, nil
}

func (db *DB) Contacts() []app.User {
	var users []user
	db.db.Preload("Messages").Find(&users)

	contacts := make([]app.User, len(users))
	for i, user := range users {
		msgs := make([]string, len(user.Messages))
		for j, msg := range user.Messages {
			msgs[j] = msg.Msg
		}
		contacts[i] = app.User{
			ID:           common.HexToAddress(user.ID),
			Name:         user.Name,
			AppLastNonce: user.AppLastNonce,
			LastNonce:    user.LastNonce,
			Key:          user.Key,
			Messages:     msgs,
		}
	}
	return contacts
}

func (db *DB) InsertMessage(id common.Address, msg string) error {
	res := db.db.Create(&message{
		Msg:    msg,
		UserID: id.Hex(),
	})

	if res.Error != nil {
		return fmt.Errorf("insert message: %w", res.Error)
	}

	return nil
}

func (db *DB) UpdateAppNonce(id common.Address, nonce uint64) error {
	res := db.db.Model(&user{}).Where("LOWER(id) = LOWER(?)", id.Hex()).Update("app_last_nonce", nonce)
	if res.Error != nil {
		return fmt.Errorf("update app nonce: %w", res.Error)
	}
	return nil
}

func (db *DB) UpdateContactNonce(id common.Address, nonce uint64) error {
	res := db.db.Model(&user{}).Where("LOWER(id) = LOWER(?)", id.Hex()).Update("last_nonce", nonce)
	if res.Error != nil {
		return fmt.Errorf("update contact nonce: %w", res.Error)
	}
	return nil
}

func (db *DB) UpdateContactKey(id common.Address, key string) error {
	res := db.db.Model(&user{}).Where("LOWER(id) = LOWER(?)", id.Hex()).Update("key", key)
	if res.Error != nil {
		return fmt.Errorf("update contact key: %w", res.Error)
	}
	return nil
}

func (db *DB) CleanTables() error {
	if err := db.db.Migrator().DropTable(&user{}, &message{}); err != nil {
		return fmt.Errorf("drop table: %w", err)
	}

	if err := db.db.AutoMigrate(&user{}, &message{}); err != nil {
		return fmt.Errorf("auto migrate: %w", err)
	}

	return nil
}
