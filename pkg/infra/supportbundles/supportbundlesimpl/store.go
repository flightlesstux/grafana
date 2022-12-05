package supportbundlesimpl

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/grafana/grafana/pkg/infra/kvstore"
	"github.com/grafana/grafana/pkg/infra/supportbundles"
)

func newStore(kv kvstore.KVStore) *store {
	return &store{kv: kvstore.WithNamespace(kv, 0, "supportbundle")}

}

type store struct {
	kv *kvstore.NamespacedKVStore
}

func (s *store) Create(ctx context.Context) (*supportbundles.Bundle, error) {
	uid, err := uuid.NewRandom()
	if err != nil {
		return nil, err
	}

	bundle := supportbundles.Bundle{
		UID:       uid.String(),
		State:     supportbundles.StatePending,
		Creator:   "kalle",
		CreatedAt: time.Now().Unix(),
		ExpiresAt: time.Now().Add(10 * time.Hour).Unix(),
	}

	if err := s.set(ctx, &bundle); err != nil {
		return nil, err
	}
	return &bundle, nil
}

func (s *store) Update(ctx context.Context, uid string, state supportbundles.State, filePath string) error {
	bundle, err := s.Get(ctx, uid)
	if err != nil {
		return err
	}

	bundle.State = state
	bundle.FilePath = filePath

	return s.set(ctx, bundle)
}

func (s *store) set(ctx context.Context, bundle *supportbundles.Bundle) error {
	data, err := json.Marshal(&bundle)
	if err != nil {
		return err
	}
	return s.kv.Set(ctx, bundle.UID, string(data))
}

func (s *store) Get(ctx context.Context, uid string) (*supportbundles.Bundle, error) {
	data, ok, err := s.kv.Get(ctx, uid)
	if err != nil {
		return nil, err
	}
	if !ok {
		// FIXME: handle not found
		return nil, errors.New("not found")
	}
	var b supportbundles.Bundle
	if err := json.NewDecoder(strings.NewReader(data)).Decode(&b); err != nil {
		return nil, err
	}

	return &b, nil
}

func (s *store) List() ([]supportbundles.Bundle, error) {
	data, err := s.kv.GetAll(context.Background())
	if err != nil {
		return nil, err
	}

	var res []supportbundles.Bundle
	for _, items := range data {
		for _, s := range items {
			var b supportbundles.Bundle
			if err := json.NewDecoder(strings.NewReader(s)).Decode(&b); err != nil {
				return nil, err
			}
			res = append(res, b)
		}
	}
	return res, nil
}
