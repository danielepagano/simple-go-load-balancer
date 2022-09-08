package security

// ServerSecurityConfig makes some assumptions on how we store certificates, and lets you change some values
type ServerSecurityConfig struct {
	ClientsCertPath   string // Base path of client certificates
	ClientCertFileExt string // Extension of client cert files, including dot, e.g. ".crt"
	ClientCertKeyExt  string // Extension of client key files, including dot, e.g. ".key"
	CaCert            string // path to CA cert file
	ServerCert        string // path to server cert file
	ServerKey         string // path to server cert key
}
