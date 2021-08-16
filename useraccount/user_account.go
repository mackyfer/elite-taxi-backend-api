package useraccount

import (
	"fmt"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//UserAccount stores user account data.
type UserAccount struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	PhoneNumber string
	Name        string
	Address     string
	Status      int
}

const connstr = "localhost:27017"

//UserExists checks the database if a user's data is present in the dataabase.
func (u *UserAccount) UserExists(phoneNumber string) bool {
	session, err := mgo.Dial(connstr)
	if err != nil {
		//log.Fatal(err)
		return false
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("user_account")
	result := UserAccount{}
	//bson.ObjectIdHex()

	err = c.Find(bson.M{"phonenumber": phoneNumber}).One(&result)
	if err != nil {
		return false
	}

	return true
}

//GetUser retrieves the userinformation from database. and stores it in the struct.
func (u *UserAccount) GetUser(id string) {
	user := UserAccount{}
	session, err := mgo.Dial(connstr)
	if err != nil {
		u = &UserAccount{}
		//return UserAccount{}
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("elite-taxi-app").C("user_account")
	if bson.IsObjectIdHex(id) {
		err = c.FindId(bson.ObjectIdHex(id)).One(&user)

		if err != nil {
			u = &UserAccount{}
		}
	} else {
		u = &UserAccount{}
	}

	*u = user
	//fmt.Println("OUT DATA:", *u)
}

//AddAccount adds a user account to the database
func (u *UserAccount) AddAccount() {
	session, err := mgo.Dial(connstr)
	if err != nil {
		//log.Fatal(err)
		return
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("user_account")
	index := mgo.Index{
		Key:        []string{"phonenumber"},
		Unique:     true,
		DropDups:   true,
		Background: true, // See notes.
		Sparse:     true,
	}
	c.EnsureIndex(index)
	err = c.Insert(&UserAccount{PhoneNumber: u.PhoneNumber, Name: u.Name, Address: u.Address, Status: u.Status})
	if err != nil {
		fmt.Println("Error: ", err)

	}
}
