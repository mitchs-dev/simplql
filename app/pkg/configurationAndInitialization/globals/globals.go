package globals

// Config vars
var (
	SimplQLIdPlaceholder  = "$ThisShouldBeReplaced"
	UseJWT                bool
	ConfigFile            string
	DefaultConfigFileName = "default.yaml"
	ApplicationName       = "SimplQL"
	EncryptionKey         string
	EncryptionIVString    string
	EncryptionIV          []byte
	EncryptionKeyFile     = "key"
	// UserPasswordLength is the length of the user password
	UserPasswordLength = 32
	UserIDLength       = 6
	ColumnTypes        = []string{
		"CHAR(n)",
		"VARCHAR(n)",
		"TEXT",
		"TINYTEXT",
		"MEDIUMTEXT",
		"LONGTEXT",
		"NCHAR(n)",
		"NVARCHAR(n)",
		"NTEXT",
		"BLOB",
		"CLOB (Character Large Object)",
		"NCLOB",
	}
)

// Built-in DB table vars
var (
	SystemTablePrefix     = "__"
	SystemParameterPrefix = "__"
	SystemColumnPrefix    = "sys_"
	SystemRolePrefix      = "__db:"
	SystemUserID          = SystemRolePrefix + "system"
	MetadataTable         = SystemTablePrefix + "metadata"
	UsersTable            = SystemTablePrefix + "users"
	JWTTable              = SystemTablePrefix + "jwts"
	TransactionsTable     = SystemTablePrefix + "transactions"
	RolesSystemAdmin      = SystemRolePrefix + "admin"
	RolesSystemUser       = SystemRolePrefix + "user"
	RolesSystemReadOnly   = SystemRolePrefix + "readonly"
	DefaultRoles          = []string{RolesSystemAdmin}
)

// JWT vars
var (
	JWTTimeZone         = "Local"
	JWTTimeoutPeriod    string
	JWTIssuer           = ApplicationName + " Server (" + SimplQLIdPlaceholder + ")"
	JWTRandomDataLength = 32
)

// Request vars
var (
	RequestSchemaData         []byte
	RequestSchemaFileName     = "requestSchema.yaml"
	NetworkingAPIBasePath     = "/api"
	NetworkingAPIVersion      = "v1"
	NetworkingAPIEndpoint     = NetworkingAPIBasePath + "/" + NetworkingAPIVersion
	NetworkingRequestAuthPath = NetworkingAPIEndpoint + "/auth"
)

// Table vars
var (
	TableEntryIDLength     = 32
	TableEntryIDPrefix     = "eid::"
	TableEntryIDSuffix     = "::eid"
	TableEntryIDColumnName = SystemColumnPrefix + "eid"

	UserEntryIDColumnName  = "id"
	UserNameColumnName     = "name"
	UserPasswordColumnName = "password"
	UserRolesColumnName    = "roles"
	RequestSelectParameter = SystemParameterPrefix + "select"
	RequestUpdateParameter = SystemParameterPrefix + "update"
	IsTransactionExecution bool
)

// Headers and Environment vars
var (

	// Environment vars
	EncryptionKeyEnvironmentVariable          = "SIMPLQL_ENCRYPTION_KEY"
	SessionDefaultUsernameEnvironmentVariable = "SIMPLQL_DEFAULT_NAME"
	SessionDefaultPasswordEnvironmentVariable = "SIMPLQL_DEFAULT_PASSWORD"
	GlobalDevelopmentBuildEnvironmentVariable = "SIMPLQL_DEV_BUILD"

	// Headers
	NetworkingHeaderHealthZ                       = "X-Healthz"
	NetworkingHeaderCorrelationID                 = "X-Correlation-ID"
	AuthenticationHeaderJWTSessionToken           = "X-JWT-Token"
	AuthenticationHeaderSessionTimeout            = "X-Session-Timeout"
	AuthenticationAuthorizationHeader             = "Authorization"
	AuthenticationAuthorizationHeaderBasicPrefix  = "Basic "
	AuthenticationAuthorizationHeaderBearerPrefix = "Bearer "
)

// Encryption vars
var (
	EncryptionOriginalFormatHeaderStart = "__ORF::"
	EncryptionOriginalFormatHeaderEnd   = "::ORF__"
	EncryptionOriginalFormatVar         = "$ORGINAL_FORMAT"
	EncryptionOriginalFormatHeader      = EncryptionOriginalFormatHeaderStart + EncryptionOriginalFormatVar + EncryptionOriginalFormatHeaderEnd
)

// Error messages
var (
	ErrorDataProcessing                     = "PROCESSING_ERROR"
	ErrorValidatingOriginalFormatHeader     = "VALIDATION_ERROR"
	ErrorInvalidColumnType                  = "INVALID_COLUMN_TYPE"
	ErrorInvalidTableName                   = "INVALID_TABLE_NAME"
	ErrorDatabaseInitialization             = "DATABASE_INITIALIZATION_ERROR"
	ErrorTransaction                        = "TRANSACTION_ERROR"
	ErrorTransactionTableNameExtraction     = "TRANSACTION_TABLE_NAME_EXTRACTION"
	ErrorTransactionRecordIDExtraction      = "TRANSACTION_RECORD_ID_EXTRACTION"
	ErrorTransactionNoEntry                 = "TRANSACTION_NO_ENTRY"
	ErrorJWTDisabled                        = "JWT_DISABLED"
	ErrorNotExist                           = "DOES_NOT_EXIST"
	ErrorAuthenticationNoRoles              = "AUTH_NO_ROLES"
	ErrorAuthenticationInvalid              = "AUTH_INVALID"
	ErrorAuthenticationUserNotFound         = "AUTH_USER_NOT_FOUND"
	ErrorAuthenticationJWTExpired           = "AUTH_JWT_EXPIRED"
	ErrorAuthenticationJWTExpiredFromJWTLib = "token invalid: expired - Login required to refresh token" // This is the specific error from mitchs-dev/library-go/jwt
)
