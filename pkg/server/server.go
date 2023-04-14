package server

type Server interface {
	//Get number of Connections made per sec to server
	GetNumConnPerSec() int
	//Get number of Active Connections to server
	GetNumActiveConn() int
	//Get number of Connections ever made to server
	GetNumTotalConn() int
	//Get unique ip per sec
	GetIpPerSec() int
	//Start the server
	Start() error
	//Stop the server
	Stop() error
}
