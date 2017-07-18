package mysql

//https://mariadb.com/kb/en/mariadb/1-connecting-connecting/

const (
	CLIENT_MYSQL    = 1 << 0
	FOUND_ROWS      = 1 << 1
	CONNECT_WITH_DB = 1 << 3
	COMPRESS        = 1 << 5
	LOCAL_FILES     = 1 << 7
	IGNORE_SPACE    = 1 << 8
	// CLIENT_PROTOCOL_41                  = 1 << 9
	// CLIENT_INTERACTIVE                  = 1 << 10
	SSL                                 = 1 << 11
	TRANSACTIONS                        = 1 << 12
	SECURE_CONNECTION                   = 1 << 13
	MULTI_STATEMENTS                    = 1 << 16
	MULTI_RESULTS                       = 1 << 17
	PS_MULTI_RESULTS                    = 1 << 18
	PLUGIN_AUTH                         = 1 << 19
	CONNECT_ATTRS                       = 1 << 20
	PLUGIN_AUTH_LENENC_CLIENT_DATA      = 1 << 21
	CLIENT_SESSION_TRACK                = 1 << 23
	CLIENT_DEPRECATE_EOF                = 1 << 24
	MARIADB_CLIENT_PROGRESS             = 1 << 32
	MARIADB_CLIENT_COM_MULTI            = 1 << 33
	MARIADB_CLIENT_STMT_BULK_OPERATIONS = 1 << 34
)
