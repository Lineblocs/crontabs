package repository

import (
	helpers "github.com/Lineblocs/go-helpers"
)

type WorkspaceRepository interface {
	GetWorkspaceFromDB(id int) (*helpers.Workspace, error)
	GetUserFromDB(id int) (*helpers.User, error)
}

type WorkspaceService struct{}

func NewWorkspaceService() WorkspaceRepository {
	return &WorkspaceService{}
}

func (ws *WorkspaceService) GetWorkspaceFromDB(id int) (*helpers.Workspace, error) {
	return helpers.GetWorkspaceFromDB(id)
}

func (ws *WorkspaceService) GetUserFromDB(id int) (*helpers.User, error) {
	return helpers.GetUserFromDB(id)
}
