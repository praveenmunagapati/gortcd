package turn

import (
	"time"

	"github.com/gortc/stun"
)

// DefaultLifetime in RFC 5766 is 10 minutes.
//
// RFC 5766 Section 2.2
const DefaultLifetime = time.Minute * 10

// Lifetime represents LIFETIME attribute.
//
// The LIFETIME attribute represents the duration for which the server
// will maintain an allocation in the absence of a refresh. The value
// portion of this attribute is 4-bytes long and consists of a 32-bit
// unsigned integral value representing the number of seconds remaining
// until expiration.
//
// RFC 5766 Section 14.2
type Lifetime struct {
	time.Duration
}

// uint32 seconds
const lifetimeSize = 4 // 4 bytes, 32 bits

// AddTo adds LIFETIME to message.
func (l Lifetime) AddTo(m *stun.Message) error {
	v := make([]byte, lifetimeSize)
	bin.PutUint32(v, uint32(l.Seconds()))
	m.Add(stun.AttrLifetime, v)
	return nil
}

// GetFrom decodes LIFETIME from message.
func (l *Lifetime) GetFrom(m *stun.Message) error {
	v, err := m.Get(stun.AttrLifetime)
	if err != nil {
		return err
	}
	if len(v) != lifetimeSize {
		return &BadAttrLength{
			Attr:     stun.AttrLifetime,
			Got:      len(v),
			Expected: lifetimeSize,
		}
	}
	_ = v[lifetimeSize-1] // asserting length
	seconds := bin.Uint32(v)
	l.Duration = time.Second * time.Duration(seconds)
	return nil
}

// ZeroLifetime is shorthand for setting zero lifetime
// that indicates to close allocation.
var ZeroLifetime stun.Setter = Lifetime{}
