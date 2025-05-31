package cabrequest

import (
	"encoding/json" // Added encoding/json
	"fmt"
	"log"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"apiserver/config" // Added config import
)

//CabRequest stores information about passenger cab requests.
type CabRequest struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	From        string        `json:"from"`
	To          string        `json:"to"`
	RequestedBy string        `json:"requestedBy"`
	ETA         string        `json:"eta"`
	Status      int           `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
}

// CabRequestOutput is used for custom JSON marshalling, especially for ObjectId.
type CabRequestOutput struct {
	ID          string    `json:"id"`
	From        string    `json:"from"`
	To          string    `json:"to"`
	RequestedBy string    `json:"requestedby"`
	ETA         string    `json:"eta"`
	Status      int       `json:"status"`
	CreatedAt   time.Time `json:"createdAt"` // time.Time marshals to RFC3339 by default
}

// CancelRequest cancels a request for a given id.
func (r *CabRequest) CancelRequest(requestIDHex string) error {
	if !bson.IsObjectIdHex(requestIDHex) {
		return fmt.Errorf("invalid request ID format: %s", requestIDHex)
	}
	objectID := bson.ObjectIdHex(requestIDHex)

	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return err
	}
	defer session.Close()
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("elite-taxi-app").C("cab_request")

	// Use objectID in your query
	where := bson.M{"_id": objectID, "status": bson.M{"$eq": 1}}
	change := bson.M{"$set": bson.M{"status": 0}}
	err = c.Update(where, change)
	if err != nil {
		if err == mgo.ErrNotFound {
			return fmt.Errorf("active request with ID %s not found", requestIDHex)
		}
		log.Printf("Error updating cab request status: %v", err)
		return err
	}
	return nil
}

// GetRequestDetail retreives information about a cab request.
func (r *CabRequest) GetRequestDetail(phone string) error {
	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error dialing MongoDB: %v", err)
		return err
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("cab_request")
	where := bson.M{"requestedby": phone, "status": 1}
	err = c.Find(where).One(r) // Populate r directly
	if err != nil {
		log.Printf("Error finding cab request: %v", err)
		if err == mgo.ErrNotFound {
			return fmt.Errorf("request not found for phone %s", phone)
		}
		return err
	}
	return nil
}

// ToString now returns marshaled JSON data for the request.
func (r *CabRequest) ToString() ([]byte, error) { // Changed signature
	output := CabRequestOutput{
		ID:          r.ID.Hex(), // Convert ObjectId to hex string
		From:        r.From,
		To:          r.To,
		RequestedBy: r.RequestedBy,
		ETA:         r.ETA,
		Status:      r.Status,
		CreatedAt:   r.CreatedAt,
	}
	return json.Marshal(output)
}

//AddCabRequest adds a user request for a cab in the database.
func (r *CabRequest) AddCabRequest() error {
	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error dialing MongoDB: %v", err)
		return err
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("cab_request")

	r.ID = bson.NewObjectId() // Ensure ID is generated before insert
	err = c.Insert(r)
	if err != nil {
		log.Printf("Error inserting cab request: %v", err)
		return err
	}

	return nil
}
