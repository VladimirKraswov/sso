package main

import (
	"errors"
	"flag"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/sqlite3"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main () {
	var storagePath, migrationsPath, migrationsTable string

	flag.StringVar(&storagePath, "storage-path", "", "path to storage")
	flag.StringVar(&migrationsPath, "migrations-path", "", "path to migrations")
	flag.StringVar(&migrationsTable, "migrations-table", "migrations", "name of migrations table")
	flag.Parse()

	if storagePath == "" {
		panic("storage-path is required")
	}
	if migrationsPath == "" {
		panic("migration-path is required")
	}

	// создаем экземпляр мигратора
	m, err := migrate.New("file://"+migrationsPath, fmt.Sprintf("sqlite3://%s?x-migrations-table=%s", storagePath, migrationsTable))
	if err != nil {
		panic(err)
	}
	// Выполняем миграцию
	// Up просматривает текущую активную версию миграции и выполняет полную миграцию вверх (применяя все миграции вверх).
	if err := m.Up(); err != nil {
		if errors.Is(err, migrate.ErrNoChange) { // Если в миграциях нет изменений, все миграции применены и применять нечего
			fmt.Println("no migrations to apply")
			return
		}
		// Если это какая-то другая ошибка то вызовем панику
		panic(err)
	}

	// Все ок, миграции применили
	fmt.Println("migrations applied successfully")

}