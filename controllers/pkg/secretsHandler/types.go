package secretsHandler

type Secret struct {
	Name  string
	Value string
}

type SecretList struct {
	Secrets []Secret
}
