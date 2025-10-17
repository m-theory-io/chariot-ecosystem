package tests

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
	cfg "github.com/bhouse1273/chariot-ecosystem/services/go-chariot/configs"
	"github.com/bhouse1273/kissflag"
)

var (
	mySQLConfigOnce sync.Once
)

func initMySQLConfig() {
	mySQLConfigOnce.Do(func() {
		// Initialize configuration similar to cmd/main.go
		// Set the env vars in the os
		os.Setenv("CHARIOT_SQL_DRIVER", "mysql")
		os.Setenv("CHARIOT_SQL_URL", "localhost")
		os.Setenv("CHARIOT_SQL_USER", "chariot")
		os.Setenv("CHARIOT_SQL_PASSWORD", "chariot123")
		os.Setenv("CHARIOT_SQL_DATABASE", "testsql")
		os.Setenv("CHARIOT_SQL_PORT", "3306")
		os.Setenv("CHARIOT_DATA_PATH", "~/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot/tests/data")
		os.Setenv("CHARIOT_TREE_PATH", "~/go/src/github.com/bhouse1273/chariot-ecosystem/services/go-chariot/tests/data/tree")
		os.Setenv("CHARIOT_VAULT_NAME", "chariot-vault")
		kissflag.SetPrefix("CHARIOT_")

		// Read configuration from environment variables
		cfg.ChariotConfig.BoolVar("headless", &cfg.ChariotConfig.Headless, false)
		cfg.ChariotConfig.IntVar("port", &cfg.ChariotConfig.Port, 8080)
		cfg.ChariotConfig.IntVar("timeout", &cfg.ChariotConfig.Timeout, 30)
		cfg.ChariotConfig.BoolVar("verbose", &cfg.ChariotConfig.Verbose, false)

		// MySQL specific configuration
		cfg.ChariotConfig.StringVar("sql_driver", &cfg.ChariotConfig.SQLDriver, "mysql")
		cfg.ChariotConfig.StringVar("sql_host", &cfg.ChariotConfig.SQLHost, "localhost")
		cfg.ChariotConfig.StringVar("sql_user", &cfg.ChariotConfig.SQLUser, "chariot")
		cfg.ChariotConfig.StringVar("sql_password", &cfg.ChariotConfig.SQLPassword, "chariot123")
		cfg.ChariotConfig.StringVar("sql_database", &cfg.ChariotConfig.SQLDatabase, "testsql")
		cfg.ChariotConfig.IntVar("sql_port", &cfg.ChariotConfig.SQLPort, 3306)

		cfg.ChariotConfig.StringVar("vault_name", &cfg.ChariotConfig.VaultName, "chariot-vault")

		// Bind environment variables
		kissflag.BindAllEVars(cfg.ChariotConfig)
	})
}

func TestMySQLOperations(t *testing.T) {
	initMySQLConfig()
	// Load config from environment or use defaults
	mysqlURL, mysqlUser, mysqlPassword, mysqlDatabase := cfg.ChariotConfig.SQLHost, cfg.ChariotConfig.SQLUser, cfg.ChariotConfig.SQLPassword, cfg.ChariotConfig.SQLDatabase
	tests := []TestCase{
		{
			Name: "Connect to MySQL",
			Script: []string{
				fmt.Sprintf(`sqlConnect('mysql-test', '%s', '%s', '%s', '%s')`, mysqlURL, mysqlUser, mysqlPassword, mysqlDatabase),
			},
			ExpectedValue: chariot.Str("Connected to mysql database"),
		},
		{
			Name: "Create Table",
			Script: []string{
				`sqlExecute('mysql-test', 'CREATE TABLE IF NOT EXISTS chariot_test (id INT PRIMARY KEY AUTO_INCREMENT, name VARCHAR(64))')`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Insert Row",
			Script: []string{
				`sqlExecute('mysql-test', 'INSERT INTO chariot_test (name) VALUES ("Alice")')`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Query Row",
			Script: []string{
				`setq(results, sqlQuery('mysql-test', 'SELECT name FROM chariot_test WHERE name = "Alice"'))`,
				`setq(row, getAt(results, 0))`,
				`getProp(row, 'name')`,
			},
			ExpectedValue: chariot.Str("Alice"),
		},
		{
			Name: "Delete Row",
			Script: []string{
				`sqlExecute('mysql-test', 'DELETE FROM chariot_test WHERE name = "Alice"')`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Drop Table",
			Script: []string{
				`sqlExecute('mysql-test', 'DROP TABLE IF EXISTS chariot_test')`,
			},
			ExpectedValue: chariot.Number(1),
		},
		{
			Name: "Close Connection",
			Script: []string{
				`sqlClose('mysql-test')`,
			},
			ExpectedValue: chariot.Str("SQL connection closed"),
		},
	}

	RunTestCases(t, tests)
}
