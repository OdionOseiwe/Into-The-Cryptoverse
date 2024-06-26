package keystore

import (
	"context"
	"fmt"

	"github.com/pkg/errors"

	"github.com/smartcontractkit/chainlink/v2/core/services/keystore/keys/dkgsignkey"
)

//go:generate mockery --quiet --name DKGSign --output mocks/ --case=underscore

// DKGSign provides signing keys for the DKG.
type DKGSign interface {
	Get(id string) (dkgsignkey.Key, error)
	GetAll() ([]dkgsignkey.Key, error)
	Create(ctx context.Context) (dkgsignkey.Key, error)
	Add(ctx context.Context, key dkgsignkey.Key) error
	Delete(ctx context.Context, id string) (dkgsignkey.Key, error)
	Import(ctx context.Context, keyJSON []byte, password string) (dkgsignkey.Key, error)
	Export(id string, password string) ([]byte, error)
	EnsureKey(ctx context.Context) error
}

type dkgSign struct {
	*keyManager
}

func newDKGSignKeyStore(km *keyManager) *dkgSign {
	return &dkgSign{
		keyManager: km,
	}
}

var _ DKGSign = &dkgSign{}

// Add implements DKGSign
func (d *dkgSign) Add(ctx context.Context, key dkgsignkey.Key) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.isLocked() {
		return ErrLocked
	}
	return d.safeAddKey(ctx, key)
}

// Create implements DKGSign
func (d *dkgSign) Create(ctx context.Context) (dkgsignkey.Key, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.isLocked() {
		return dkgsignkey.Key{}, ErrLocked
	}
	key, err := dkgsignkey.New()
	if err != nil {
		return dkgsignkey.Key{}, errors.Wrap(err, "dkgsignkey New()")
	}
	return key, d.safeAddKey(ctx, key)
}

// Delete implements DKGSign
func (d *dkgSign) Delete(ctx context.Context, id string) (dkgsignkey.Key, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.isLocked() {
		return dkgsignkey.Key{}, ErrLocked
	}
	key, err := d.getByID(id)
	if err != nil {
		return dkgsignkey.Key{}, err
	}

	err = d.safeRemoveKey(ctx, key)
	return key, errors.Wrap(err, "safe remove key")
}

// EnsureKey implements DKGSign
func (d *dkgSign) EnsureKey(ctx context.Context) error {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.isLocked() {
		return ErrLocked
	}
	if len(d.keyRing.DKGSign) > 0 {
		return nil
	}

	key, err := dkgsignkey.New()
	if err != nil {
		return errors.Wrap(err, "dkgsignkey New()")
	}

	d.logger.Infof("Created DKGSign key with ID %s", key.ID())

	return d.safeAddKey(ctx, key)
}

// Export implements DKGSign
func (d *dkgSign) Export(id string, password string) ([]byte, error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.isLocked() {
		return nil, ErrLocked
	}
	key, err := d.getByID(id)
	if err != nil {
		return nil, err
	}
	return key.ToEncryptedJSON(password, d.scryptParams)
}

// Get implements DKGSign
func (d *dkgSign) Get(id string) (keys dkgsignkey.Key, err error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.isLocked() {
		return dkgsignkey.Key{}, ErrLocked
	}
	return d.getByID(id)
}

// GetAll implements DKGSign
func (d *dkgSign) GetAll() (keys []dkgsignkey.Key, err error) {
	d.lock.RLock()
	defer d.lock.RUnlock()
	if d.isLocked() {
		return nil, ErrLocked
	}
	for _, key := range d.keyRing.DKGSign {
		keys = append(keys, key)
	}
	return keys, nil
}

// Import implements DKGSign
func (d *dkgSign) Import(ctx context.Context, keyJSON []byte, password string) (dkgsignkey.Key, error) {
	d.lock.Lock()
	defer d.lock.Unlock()
	if d.isLocked() {
		return dkgsignkey.Key{}, ErrLocked
	}
	key, err := dkgsignkey.FromEncryptedJSON(keyJSON, password)
	if err != nil {
		return dkgsignkey.Key{}, errors.Wrap(err, "from encrypted json")
	}
	_, err = d.getByID(key.ID())
	if err == nil {
		return dkgsignkey.Key{}, fmt.Errorf("key with ID %s already exists", key.ID())
	}
	return key, d.keyManager.safeAddKey(ctx, key)
}

// caller must hold lock
func (d *dkgSign) getByID(id string) (dkgsignkey.Key, error) {
	key, found := d.keyRing.DKGSign[id]
	if !found {
		return dkgsignkey.Key{}, KeyNotFoundError{
			ID:      id,
			KeyType: "DKGSign",
		}
	}
	return key, nil
}
