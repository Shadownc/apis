package db

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

func ConnectAndCreateDBIfNotExist(username, password, host, port, dbname string) (*sql.DB, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port, dbname)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		if err.Error() == fmt.Sprintf("Error 1049 (42000): Unknown database '%s'", dbname) {
			fmt.Println("Database not found, creating...")
			db, err := sql.Open("mysql", fmt.Sprintf("%s:%s@tcp(%s:%s)/?charset=utf8mb4&parseTime=True&loc=Local",
				username, password, host, port))
			if err != nil {
				return nil, err
			}

			_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbname))
			if err != nil {
				return nil, err
			}

			db.Close()

			// Reconnect to the newly created database
			db, err = sql.Open("mysql", dsn)
			if err != nil {
				return nil, err
			}
			err = db.Ping()
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	fmt.Println("Database connected successfully!")
	return db, nil
}

func CreateTableIfNotExists(db *sql.DB, tableName string) error {
	createTableQuery := fmt.Sprintf(`
    CREATE TABLE IF NOT EXISTS %s (
        id INT AUTO_INCREMENT PRIMARY KEY,
        api_url VARCHAR(255) NOT NULL,
        request_params TEXT,
        response_data TEXT,
        call_count INT DEFAULT 1,
        created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
        updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
    );`, tableName)

	_, err := db.Exec(createTableQuery)
	if err != nil {
		return err
	}

	fmt.Println("Table checked/created successfully!")
	return nil
}
