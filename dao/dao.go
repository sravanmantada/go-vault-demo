package dao

import (
	"encoding/base64"
	"log"
	"strconv"
	"time"

	"github.com/go-pg/pg"
	"github.com/go-pg/pg/orm"
	. "github.com/lanceplarsen/go-vault-demo/client"
	. "github.com/lanceplarsen/go-vault-demo/models"
)

type OrderDAO struct {
	Url      string
	Database string
	User     string
	Password string
	Vault    *Vault
}

var db *pg.DB

func (o *OrderDAO) Connect() error {
	var n int

	//conn string
	db = pg.Connect(&pg.Options{
		User:     o.User,
		Password: o.Password,
		Addr:     o.Url,
		Database: o.Database,
	})

	//Check our connection
	_, err := db.QueryOne(pg.Scan(&n), "SELECT 1")
	return err
}

func (o *OrderDAO) Close() error {
	err := db.Close()
	return err

}

func (o *OrderDAO) FindAll() ([]Order, error) {
	var eOrders []Order
	var dOrders []Order

	//Go get the orders
	err := db.Model(&eOrders).Select()
	if err != nil {
		return []Order{}, err
	}

	//Decrypt these. TODO Could use a batch decyrpt opp here
	for _, order := range eOrders {
		dOrder, err := o.Vault.Decrypt(order.CustomerName)
		if err != nil {
			log.Println("Unable to decrypt order: " + strconv.FormatInt(order.Id, 10))
		} else {
			sDec, _ := base64.StdEncoding.DecodeString(dOrder)
			order.CustomerName = string(sDec)
			dOrders = append(dOrders, order)
		}
	}

	return dOrders, nil
}

func (o *OrderDAO) DeleteAll() error {
	var ids []int
	var res orm.Result

	//Find the order ids
	err := db.Model(&Order{}).Column("id").Select(&ids)
	if err != nil {
		return err
	}
	//Delete the order ids if we have results
	if len(ids) > 0 {
		pgids := pg.In(ids)
		res, err = db.Model(&Order{}).Where("id IN (?)", pgids).Delete()
		if err != nil {
			return err
		}
		log.Println("Deleted records", res.RowsAffected())
	} else {
		log.Println("No records to delete.")
	}
	return nil
}

func (o *OrderDAO) Insert(order Order) (Order, error) {
	//Add a timestamp
	order.OrderDate = time.Now()

	//Encrypt it
	encode := base64.StdEncoding.EncodeToString([]byte(order.CustomerName))
	//Get plaintext customer
	cipher, err := o.Vault.Encrypt(encode)
	if err != nil {
		return order, err
	}

	//Insert the order
	order.CustomerName = cipher
	err = db.Insert(&order)
	if err != nil {
		return order, err
	}

	return order, nil
}
