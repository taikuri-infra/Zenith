package entities

// DatabaseEngine represents the type of database engine.
type DatabaseEngine string

const (
	DatabaseEnginePostgres DatabaseEngine = "postgresql"
	DatabaseEngineMySQL    DatabaseEngine = "mysql"
	DatabaseEngineRedis    DatabaseEngine = "redis"
)

// DatabaseStatus represents the lifecycle status of a provisioned database.
type DatabaseStatus string

const (
	DatabaseStatusProvisioning DatabaseStatus = "provisioning"
	DatabaseStatusReady        DatabaseStatus = "ready"
	DatabaseStatusError        DatabaseStatus = "error"
	DatabaseStatusDeleting     DatabaseStatus = "deleting"
)

// UserDatabase represents a database provisioned for a user's app.
type UserDatabase struct {
	ID        string         `json:"id"`
	AppID     string         `json:"app_id"`
	UserID    string         `json:"user_id"`
	Name      string         `json:"name"`
	Engine    DatabaseEngine `json:"engine"`
	DBName    string         `json:"db_name"`
	DBUser    string         `json:"db_user"`
	Host      string         `json:"host"`
	Port      int            `json:"port"`
	SizeMB    int            `json:"size_mb"`
	MaxSizeMB int            `json:"max_size_mb"`
	Status    DatabaseStatus `json:"status"`
	Timestamps
}

// ConnectionString returns the full DSN for this database.
func (d *UserDatabase) ConnectionString(password string) string {
	switch d.Engine {
	case DatabaseEnginePostgres:
		return "postgresql://" + d.DBUser + ":" + password + "@" + d.Host + ":" + itoa(d.Port) + "/" + d.DBName + "?sslmode=disable"
	case DatabaseEngineMySQL:
		return d.DBUser + ":" + password + "@tcp(" + d.Host + ":" + itoa(d.Port) + ")/" + d.DBName
	case DatabaseEngineRedis:
		return "redis://:" + password + "@" + d.Host + ":" + itoa(d.Port) + "/0"
	default:
		return ""
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	s := ""
	for n > 0 {
		s = string(rune('0'+n%10)) + s
		n /= 10
	}
	return s
}
