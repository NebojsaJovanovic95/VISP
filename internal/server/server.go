package server

import (
	"context"
	"fmt"

	"github.com/NebojsaJovanovic95/VISP/internal/database"
)

// Run starts the VISP Server.
func Run() error {
	ctx := context.Background()

	db, err := database.New(ctx)
	if err != nil {
		return fmt.Errorf("initialize database: %w", err)
	}
	defer db.Close()

	fmt.Println("VISP Server started")

	return nil
}
