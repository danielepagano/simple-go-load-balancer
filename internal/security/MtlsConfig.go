package security

// ServerSecurityConfig makes some assumptions on how we store certificates, and lets you change some values
type ServerSecurityConfig struct {
	EnableMutualTLS bool   // Master switch that turns off security to simplify testing in this sample project
	CertPath        string // Base path of certificates
	ClientsCertPath string // Base path of client certificates (not evaluated as a subdirectory of CertPath)
	CertFileExt     string // Extension of cert files, including dot, e.g. ".crt"
	CertKeyExt      string // Extension of key files, including dot, e.g. ".key"
	CaCertName      string // e.g. "server"
	ServerCertName  string // e.g. "ca"
}
