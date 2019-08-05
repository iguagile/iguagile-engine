package iguagile

// GameObject is object to be synchronized.
type GameObject struct {
	id           int
	owner        *Client
	resourcePath []byte
}
