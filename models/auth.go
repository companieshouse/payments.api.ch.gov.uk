package models

// AuthUserDetails is a representation of user details retrieved from the eric headers in a request
type AuthUserDetails struct {
	Email    string
	Forename string
	Surname  string
	Id       string
}
