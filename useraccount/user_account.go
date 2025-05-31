package useraccount

import (
	"fmt"
	"log"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"apiserver/config" // Added config import
)

//UserAccount stores user account data.
type UserAccount struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	PhoneNumber string
	Name        string
	Address     string
	Status      int
}

//UserExists checks the database if a user's data is present in the dataabase.
func (u *UserAccount) UserExists(phoneNumber string) (bool, error) {
	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error dialing MongoDB: %v", err)
		return false, err
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("user_account")
	result := UserAccount{}

	err = c.Find(bson.M{"phonenumber": phoneNumber}).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil // User does not exist, but no actual error
		}
		log.Printf("Error finding user: %v", err)
		return false, err
	}

	return true, nil
}

//GetUser retrieves the userinformation from database. and stores it in the struct.
func (u *UserAccount) GetUser(id string) error {
	if !bson.IsObjectIdHex(id) {
		return fmt.Errorf("invalid user ID format: %s", id)
	}

	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return err
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("user_account")

	var user UserAccount // Temporary variable to fetch data into
	objectID := bson.ObjectIdHex(id)
	err = c.FindId(objectID).One(&user)
	if err != nil {
		if err == mgo.ErrNotFound {
			return fmt.Errorf("user with ID %s not found", id)
		}
		log.Printf("Error finding user with ID %s: %v", id, err)
		return err
	}

	*u = user // Populate the receiver struct
	return nil
}

//AddAccount adds a user account to the database
func (u *UserAccount) AddAccount() error {
	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error dialing MongoDB: %v", err)
		return err
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
	// EnsureIndex can return an error, though it's often related to index creation conflicts
	// which might not be critical to return to the user in all cases.
	// For now, logging it.
	if err := c.EnsureIndex(index); err != nil {
		log.Printf("Error ensuring index: %v", err)
		// Depending on policy, you might want to return this error.
	}

	u.ID = bson.NewObjectId() // Ensure ID is generated before insert
	err = c.Insert(u)
	if err != nil {
		log.Printf("Error inserting user account: %v", err)
		return err
	}
	return nil
}
