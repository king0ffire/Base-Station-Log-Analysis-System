package lowermanager

import "webapp/service/models"

type SocketManager struct {
	Socket *models.Socket
}

func NewSocketManager() *SocketManager {
	return &SocketManager{
		Socket: models.NewSocket(),
	}
}
