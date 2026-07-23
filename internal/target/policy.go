package target

import (
	"errors"
	"net/netip"
)

var ErrAddressBlocked = errors.New("target address is blocked by network policy")

type Policy struct {
	Public         bool
	AllowLoopback  bool
	AllowPrivate   bool
	AllowLinkLocal bool
}

func (p Policy) ValidateAddress(address netip.Addr) error {
	address = address.Unmap()
	if !address.IsValid() || address.IsUnspecified() || address.IsMulticast() {
		return ErrAddressBlocked
	}

	if p.Public {
		if address.IsLoopback() || address.IsPrivate() || address.IsLinkLocalUnicast() {
			return ErrAddressBlocked
		}
		return nil
	}
	if address.IsLoopback() && !p.AllowLoopback {
		return ErrAddressBlocked
	}
	if address.IsPrivate() && !p.AllowPrivate {
		return ErrAddressBlocked
	}
	if address.IsLinkLocalUnicast() && !p.AllowLinkLocal {
		return ErrAddressBlocked
	}
	return nil
}
