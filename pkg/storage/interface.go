package storage

import "github.com/kiling91/telegram-email-assistant/pkg/types"

type Storage interface {
	AddUser(user *types.EmailUser) (types.UID, error)
	GetUser(uid types.UID) (*types.EmailUser, error)
}
