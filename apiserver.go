package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/user/apiserver/taxi"
	"github.com/user/apiserver/useraccount"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

const emptyData = `{"data":{"status":0}}`
const successData = `{"data":{"status":1}}`
const connstr = "localhost:27017"

func readFile(f string) string {
	file, err := os.Open(f)
	if err != nil {
		// handle the error here
		return ""
	}
	defer file.Close()
	// get the file size
	stat, err := file.Stat()
	if err != nil {
		return ""
	}
	// read the file
	bs := make([]byte, stat.Size())
	_, err = file.Read(bs)

	if err != nil {
		return ""
	}
	return string(bs)
}

//PhoneCode stores the code and phone number combination.
type PhoneCode struct {
	PhoneNumber      string
	VerificationCode int
}

func addVerifyPhone(phone string) (bool, int) {
	session, err := mgo.Dial(connstr)
	if err != nil {
		//log.Fatal(err)
		return false, 0
	}
	defer session.Close()

	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)

	c := session.DB("elite-taxi-app").C("phone_code_verify")
	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(9999)
	err = c.Insert(&PhoneCode{phone, code})
	if err != nil {
		//log.Fatal(err)
		return false, 0
	}
	result := PhoneCode{}
	err = c.Find(bson.M{"phonenumber": phone}).One(&result)
	if err != nil {
		log.Fatal(err)
		return false, 0
	}
	return true, code
}

// check phone number checks a phone number and code against the database to ensuure
// that the combination exists.

func validPhoneNumberCombo(phoneNumber string, verifyCode int) bool {
	session, err := mgo.Dial(connstr)
	if err != nil {
		//log.Fatal(err)
		return false
	}
	defer session.Close()
	// Optional. Switch the session to a monotonic behavior.
	session.SetMode(mgo.Monotonic, true)
	c := session.DB("elite-taxi-app").C("phone_code_verify")
	result := PhoneCode{}
	err = c.Find(bson.M{"phonenumber": phoneNumber, "verificationcode": verifyCode}).One(&result)
	if err != nil {
		//log.Fatal(err)
		return false
	}
	return true
}

func validatePhone(res http.ResponseWriter, req *http.Request) {
	pageData := ""
	phone := req.URL.Query().Get("pnumber")
	pCode, err := strconv.Atoi(req.URL.Query().Get("pcode"))

	if err != nil {
		fmt.Println("Error:", err)
	}

	if validPhoneNumberCombo(phone, pCode) {
		userAccount := useraccount.UserAccount{PhoneNumber: phone, Name: "", Address: "", Status: 1}
		userAccount.AddAccount()
		pageData = successData
	} else {
		pageData = emptyData
	}

	res.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	io.WriteString(
		res,
		pageData,
	)
}

func verifyPhone(res http.ResponseWriter, req *http.Request) {

	phone := req.URL.Query().Get("pnumber")
	verify, code := addVerifyPhone(phone)
	var pageData string
	if verify == true {
		data := map[string]map[string]int{"data": {"status": 1, "code": code}}
		dat, _ := json.Marshal(data)
		pageData = string(dat)
	} else {

		pageData = emptyData
	}

	res.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	io.WriteString(
		res,
		pageData,
	)
}

func login(res http.ResponseWriter, req *http.Request) {
	pageData := ""

	phoneNumber := req.URL.Query().Get("phoneNumber")

	u := useraccount.UserAccount{}
	if u.UserExists(phoneNumber) {
		pageData = successData
	} else {
		pageData = emptyData
	}

	res.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	io.WriteString(
		res,
		pageData,
	)

}

func requestCab(res http.ResponseWriter, req *http.Request) {
	phone := req.URL.Query().Get("phoneNumber")
	from := req.URL.Query().Get("from")
	to := req.URL.Query().Get("to")
	pageData := ""

	//fmt.Println("Actual data: ", user)
	cr := cabrequest.CabRequest{From: from, To: to, RequestedBy: phone,
		ETA: "", Status: 1, CreatedAt: time.Now()}
	err := cr.AddCabRequest()
	if !err {
		pageData = emptyData
	} else {
		pageData = successData
	}
	res.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	io.WriteString(
		res,
		pageData,
	)
}

func cancelRequest(res http.ResponseWriter, req *http.Request) {
	reqID := req.URL.Query().Get("phoneNumber")
	pageData := ""
	request := cabrequest.CabRequest{}
	err := request.CancelRequest(reqID)
	if !err {
		pageData = emptyData
	} else {
		pageData = successData
	}
	res.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	io.WriteString(
		res,
		pageData,
	)
}

func getRequestInfo(res http.ResponseWriter, req *http.Request) {
	phone := req.URL.Query().Get("phoneNumber")
	pageData := ""
	request := cabrequest.CabRequest{}
	detail := request.GetRequestDetail(phone)
	if !detail {
		pageData = emptyData
	} else {
		pageData += `{"data":{"status":1},`
		pageData += request.ToString()
		pageData += `}`
	}
	res.Header().Set(
		"Content-Type",
		"application/json; charset=utf-8",
	)
	io.WriteString(
		res,
		pageData,
	)
}

func main() {

	port := ":9000"
	http.HandleFunc("/login", login)
	http.HandleFunc("/verify", verifyPhone)
	http.HandleFunc("/validate", validatePhone)
	http.HandleFunc("/requestcab", requestCab)
	http.HandleFunc("/getrequestinfo", getRequestInfo)
	http.HandleFunc("/cancelrequest", cancelRequest)
	fmt.Println("Server running on port", port)
	http.ListenAndServe(port, nil)

}
