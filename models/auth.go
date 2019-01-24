package models

// AuthUserDetails is a representation of user details retrieved from the eric headers in a request
type AuthUserDetails struct {
	User_email    string
	User_forename string
	User_surname  string
}
