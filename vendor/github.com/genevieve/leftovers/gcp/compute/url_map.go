package compute

import "fmt"

type UrlMap struct {
	client urlMapsClient
	name   string
}

func NewUrlMap(client urlMapsClient, name string) UrlMap {
	return UrlMap{
		client: client,
		name:   name,
	}
}

func (u UrlMap) Delete() error {
	err := u.client.DeleteUrlMap(u.name)

	if err != nil {
		return fmt.Errorf("ERROR deleting url map %s: %s", u.name, err)
	}

	return nil
}

func (u UrlMap) Name() string {
	return u.name
}

func (u UrlMap) Type() string {
	return "url map"
}
