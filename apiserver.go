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

	"apiserver/config" // Added config import
)

// Helper function to send JSON responses
func sendJSONResponse(res http.ResponseWriter, statusCode int, data interface{}) {
	res.Header().Set("Content-Type", "application/json; charset=utf-8")
	res.WriteHeader(statusCode)
	if err := json.NewEncoder(res).Encode(data); err != nil {
		log.Printf("Error encoding JSON response: %v", err)
		// Fallback if JSON encoding fails, though ideally this should not happen.
		http.Error(res, `{"status":"error", "message":"Internal server error encoding response"}`, http.StatusInternalServerError)
	}
}

// Error response structure
type ErrorResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// Success response structure for simple status
type SuccessStatusResponse struct {
	Status string            `json:"status"`
	Data   map[string]int `json:"data"`
}

// Success response for data
type SuccessDataResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
}

//PhoneCode stores the code and phone number combination.
type PhoneCode struct {
	PhoneNumber      string
	VerificationCode int
}

func addVerifyPhone(phone string) (int, error) {
	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error dialing MongoDB: %v", err)
		return 0, err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("elite-taxi-app").C("phone_code_verify")

	rand.Seed(time.Now().UnixNano())
	code := rand.Intn(9999) // Generate a 4-digit code

	// Remove any existing code for this phone number to avoid conflicts if re-verifying
	_, err = c.RemoveAll(bson.M{"phonenumber": phone})
	if err != nil {
		log.Printf("Error removing existing phone code for %s: %v", phone, err)
		// Decide if this is a fatal error for the operation.
		// For now, we'll log and continue, as inserting a new code might still work.
	}

	err = c.Insert(&PhoneCode{PhoneNumber: phone, VerificationCode: code})
	if err != nil {
		log.Printf("Error inserting phone code: %v", err)
		return 0, err
	}
	// No need to find it again, we have the code.
	return code, nil
}

func validPhoneNumberCombo(phoneNumber string, verifyCode int) (bool, error) {
	session, err := mgo.Dial(config.MongoConnStr) // Replaced connstr
	if err != nil {
		log.Printf("Error dialing MongoDB: %v", err)
		return false, err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	c := session.DB("elite-taxi-app").C("phone_code_verify")
	result := PhoneCode{}
	err = c.Find(bson.M{"phonenumber": phoneNumber, "verificationcode": verifyCode}).One(&result)
	if err != nil {
		if err == mgo.ErrNotFound {
			return false, nil // Combination not found, but not a server error
		}
		log.Printf("Error finding phone number combo: %v", err)
		return false, err
	}
	return true, nil
}

func validatePhone(res http.ResponseWriter, req *http.Request) {
	phone := req.URL.Query().Get("pnumber")
	pCodeStr := req.URL.Query().Get("pcode")

	if phone == "" || pCodeStr == "" {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Message: "Phone number and code are required."})
		return
	}

	pCode, err := strconv.Atoi(pCodeStr)
	if err != nil {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Data: map[string]string{"pcode": "Invalid verification code format."}})
		return
	}

	valid, err := validPhoneNumberCombo(phone, pCode)
	if err != nil {
		sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Error validating phone number combo.", Details: err.Error()})
		return
	}

	if valid {
		userAccount := useraccount.UserAccount{PhoneNumber: phone, Name: "", Address: "", Status: 1}
		if err := userAccount.AddAccount(); err != nil {
			// Check if the error is due to duplicate phone number (which means account already exists)
			// This depends on how mgo/MongoDB driver reports unique constraint violations.
			// For a more robust check, you might need to query UserExists first or parse the error.
			// For now, assume any error from AddAccount is a server-side issue.
			sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Failed to create account.", Details: err.Error()})
			return
		}
		sendJSONResponse(res, http.StatusOK, SuccessStatusResponse{Status: "success", Data: map[string]int{"status": 1}})
	} else {
		sendJSONResponse(res, http.StatusUnauthorized, ErrorResponse{Status: "fail", Message: "Invalid phone number or verification code."})
	}
}

func verifyPhone(res http.ResponseWriter, req *http.Request) {
	phone := req.URL.Query().Get("pnumber")
	if phone == "" {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Data: map[string]string{"pnumber": "Phone number is required."}})
		return
	}

	code, err := addVerifyPhone(phone)
	if err != nil {
		sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Failed to generate verification code.", Details: err.Error()})
		return
	}

	sendJSONResponse(res, http.StatusOK, SuccessDataResponse{Status: "success", Data: map[string]interface{}{"status": 1, "code": code}})
}

func login(res http.ResponseWriter, req *http.Request) {
	phoneNumber := req.URL.Query().Get("phoneNumber")
	if phoneNumber == "" {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Data: map[string]string{"phoneNumber": "Phone number is required."}})
		return
	}

	u := useraccount.UserAccount{}
	exists, err := u.UserExists(phoneNumber)
	if err != nil {
		sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Error checking user existence.", Details: err.Error()})
		return
	}

	if exists {
		sendJSONResponse(res, http.StatusOK, SuccessStatusResponse{Status: "success", Data: map[string]int{"status": 1}})
	} else {
		sendJSONResponse(res, http.StatusNotFound, ErrorResponse{Status: "fail", Message: "User not found."})
	}
}

func requestCab(res http.ResponseWriter, req *http.Request) {
	phone := req.URL.Query().Get("phoneNumber")
	from := req.URL.Query().Get("from")
	to := req.URL.Query().Get("to")

	if phone == "" || from == "" || to == "" {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Message: "Phone number, from, and to locations are required."})
		return
	}

	cr := cabrequest.CabRequest{
		From:        from,
		To:          to,
		RequestedBy: phone,
		ETA:         "", // ETA could be calculated or set differently
		Status:      1,  // Assuming 1 means active/pending
		CreatedAt:   time.Now(),
	}

	if err := cr.AddCabRequest(); err != nil {
		sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Failed to request cab.", Details: err.Error()})
		return
	}
	sendJSONResponse(res, http.StatusOK, SuccessStatusResponse{Status: "success", Data: map[string]int{"status": 1}})
}

func cancelRequest(res http.ResponseWriter, req *http.Request) {
	reqID := req.URL.Query().Get("requestID")
	if reqID == "" {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Message: "Request ID is required."})
		return
	}

	request := cabrequest.CabRequest{}
	if err := request.CancelRequest(reqID); err != nil {
		// Check for specific error messages from CancelRequest
		if err.Error() == fmt.Sprintf("invalid request ID format: %s", reqID) {
			sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "error", Message: err.Error()})
		} else if err.Error() == fmt.Sprintf("active request with ID %s not found", reqID) {
			sendJSONResponse(res, http.StatusNotFound, ErrorResponse{Status: "error", Message: err.Error()})
		} else {
			sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Failed to cancel request.", Details: err.Error()})
		}
		return
	}
	sendJSONResponse(res, http.StatusOK, SuccessStatusResponse{Status: "success", Data: map[string]int{"status": 1}})
}

func getRequestInfo(res http.ResponseWriter, req *http.Request) {
	phone := req.URL.Query().Get("phoneNumber")
	if phone == "" {
		sendJSONResponse(res, http.StatusBadRequest, ErrorResponse{Status: "fail", Message: "Phone number is required."})
		return
	}

	request := cabrequest.CabRequest{}
	if err := request.GetRequestDetail(phone); err != nil {
		if err.Error() == fmt.Sprintf("request not found for phone %s", phone) {
			sendJSONResponse(res, http.StatusNotFound, ErrorResponse{Status: "error", Message: err.Error()})
		} else {
			sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Failed to get request details.", Details: err.Error()})
		}
		return
	}
	// Assuming ToString() is still relevant and provides the core data.
	// For a more robust JSON structure, populate a struct and marshal that.
	// For now, we adapt the existing ToString() into a new structure.
	requestJSON, err := request.ToString()
	if err != nil {
		log.Printf("Error marshalling request data: %v", err)
		sendJSONResponse(res, http.StatusInternalServerError, ErrorResponse{Status: "error", Message: "Failed to serialize request data.", Details: err.Error()})
		return
	}

	// We use json.RawMessage to embed the already-marshaled JSON from request.ToString()
	// into the "request" field of our response data.
	responseData := map[string]interface{}{
		"request": json.RawMessage(requestJSON),
	}

	sendJSONResponse(res, http.StatusOK, SuccessDataResponse{Status: "success", Data: responseData})
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
