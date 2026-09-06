package cmd

import (
	"github.com/kaliv0/homie/internal/config"
	"github.com/kaliv0/homie/internal/log"
	"github.com/kaliv0/homie/internal/storage"
)

func openDB() (*storage.Repository, error) {
	dbPath, err := config.DBPath()
	if err != nil {
		return nil, err
	}
	return storage.NewRepository(dbPath)
}

func closeDB(db *storage.Repository) {
	if closeErr := db.Close(); closeErr != nil {
		log.Logger().Println(closeErr)
	}
}
