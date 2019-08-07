package iguagile

// GameObject is object to be synchronized.
type GameObject struct {
	id           int
	owner        *Client
	lifetime     byte
	resourcePath []byte
}

// lifetime
const (
	roomExist = iota
	ownerExist
)