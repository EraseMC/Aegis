package backend

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oomph-ac/oomph/anticheat/integration/proxy"
	"github.com/sandertv/gophertunnel/minecraft"
	"github.com/sandertv/gophertunnel/minecraft/protocol/login"
)

func Dial(backup string, timeout time.Duration) proxy.DialFunc {
	return func(ctx context.Context, primary string, identity login.IdentityData, client login.ClientData, _ string) (proxy.Backend, error) {
		dialer := minecraft.Dialer{
			IdentityData:        identity,
			ClientData:          client,
			KeepXBLIdentityData: true,
			FlushRate:           -1,
			EnableBatchReading:  true,
		}

		var errs []error
		for _, address := range addresses(primary, backup) {
			conn, err := dial(ctx, &dialer, address, timeout)
			if err == nil {
				return conn, nil
			}
			errs = append(errs, fmt.Errorf("%s: %w", address, err))
		}

		return nil, fmt.Errorf("dial backend: %w", errors.Join(errs...))
	}
}

func dial(ctx context.Context, dialer *minecraft.Dialer, address string, timeout time.Duration) (*minecraft.Conn, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	return dialer.DialContext(ctx, "raknet", address)
}

func addresses(primary, backup string) []string {
	if backup == "" || backup == primary {
		return []string{primary}
	}

	return []string{primary, backup}
}
