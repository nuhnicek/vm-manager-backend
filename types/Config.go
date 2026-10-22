package types

type Config struct {
	Port            string
	MongoURI        string
	Redis           string
	ProxmoxPassword string
	ProxmoxUsername string
	ProxmoxHostURI  string
	ProxmoxNode     string
	ProxmoxURIs     []string
}
