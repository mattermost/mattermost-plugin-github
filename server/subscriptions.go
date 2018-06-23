package main

const (
	SUBSCRIPTIONS_KEY = "subscriptions"
)

type Subscriptions struct {
	Repositories map[string][]string
}

/*
func NewSubscriptionsFromKVStore(store plugin.KeyValueStore) (*Subscriptions, error) {
	var subscriptions *Subscriptions

	value, err := store.Get(SUBSCRIPTIONS_KEY)
	if err != nil {
		return nil, err
	}

	if value == nil {
		subscriptions = &Subscriptions{}
	} else {
		json.NewDecoder(bytes.NewReader(value)).Decode(&subscriptions)
	}

	return subscriptions, nil
}

func (s *Subscriptions) StoreInKVStore(store plugin.KeyValueStore) error {
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	store.Set(SUBSCRIPTIONS_KEY, b)
	return nil
}

func (s *Subscriptions) GetChannelsForRepository(repository string) []string {
	fmt.Println("rep:")
	fmt.Println(s.Repositories)
	fmt.Println(s.Repositories[repository])
	return s.Repositories[repository]
}

func (s *Subscriptions) Add(channelId string, repository string) {
	if s.Repositories == nil {
		s.Repositories = make(map[string][]string)
	}
	if value, ok := s.Repositories[repository]; ok {
		value = append(value, channelId)
		s.Repositories[repository] = value
	} else {
		s.Repositories[repository] = []string{channelId}
	}
}

func (s *Subscriptions) Remove(channelId string, repository string) {
}

func (s *Subscriptions) RemoveAll(channelId string, repository string) {
}
*/
