package cabrequest

import (
	"fmt"
	"strconv"
	"time"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

//CabRequest stores information about passenger cab requests.
type CabRequest struct {
	ID          bson.ObjectId `json:"id" bson:"_id,omitempty"`
	From        string
	To          string
	RequestedBy string
	ETA         string
	Status      int
	CreatedAt   time.Time
}

const connstr = "localhost:27017"

// CancelRequest cancels a request for a given id.
func (r *CabRequest) CancelRequest(id string) bool {
	session, err := mgo.Dial(connstr)
	if err != nil {
		//log.Fatal(err)
		return false
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("cab_request")
	//fmt.Println("Where clause:", id)

	if id != "" {
		where := bson.M{"requestedby":id, "status":bson.M{"$eq":1}}
		change := bson.M{"$set": bson.M{"status": 0}}
		err = c.Update(where, change)
		fmt.Println("ERR:: ",err)
		if err != nil {
			return false
		}
	} else {
		return false
	}
	return true
}

// GetRequestDetail retreives information about a cab request.
func (r *CabRequest) GetRequestDetail(phone string) bool {
	session, err := mgo.Dial(connstr)
	defer session.Close()
	if err == nil {
		// Optional. Switch the session to a monotonic behavior.
		session.SetMode(mgo.Monotonic, true)

		c := session.DB("elite-taxi-app").C("cab_request")
		where := bson.M{"requestedby": phone, "status": 1}
		result := CabRequest{}
		err = c.Find(where).One(&result)
		if err != nil {
			return false
		}

		r.ID = result.ID
		r.From = result.From
		r.To = result.To
		r.RequestedBy = result.RequestedBy
		r.ETA = result.ETA
		r.Status = result.Status
		r.CreatedAt = result.CreatedAt

		return true

	}
	return false

}

//ToString outputs the details of the request as a key:value string.
func (r *CabRequest) ToString() string {
	output := `"request":{`
	output += `"id":"` + string(r.ID.Hex()) + `",`
	output += `"from":"` + string(r.From) + `",`
	output += `"to":"` + string(r.To) + `",`
	output += `"requestedby":"` + string(r.RequestedBy) + `",`
	output += `"eta":"` + string(r.ETA) + `",`
	output += `"status":"` + strconv.Itoa(r.Status) + `"`
	output += `}`
	return output

}

//AddCabRequest adds a user request for a cab in the database.
func (r *CabRequest) AddCabRequest() bool {
	session, err := mgo.Dial(connstr)
	if err != nil {
		//log.Fatal(err)
		return false
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("cab_request")

	err = c.Insert(&CabRequest{From: r.From, To: r.To, RequestedBy: r.RequestedBy, ETA: r.ETA, Status: r.Status, CreatedAt: r.CreatedAt})
	if err != nil {
		fmt.Println("Error: ", err)
		return false
	}

	return true
}
