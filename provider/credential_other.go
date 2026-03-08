//go:build !darwin

package provider

func credentialLoad(service, account string) (string, error) {
	return loadFromFile(service, account)
}

func credentialStore(service, account, value string) error {
	return storeToFile(service, account, value)
}
