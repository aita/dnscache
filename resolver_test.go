package dnscache

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupIP(t *testing.T) {
	t.Run("w/o cache", func(t *testing.T) {
		tests := []struct {
			name string
		}{
			{"google.com"},
			{"yahoo.co.jp"},
		}

		var r Resolver
		for _, tt := range tests {
			ips, err := r.LookupIP(context.Background(), tt.name)
			if err != nil {
				t.Fatal(err)
			}
			assert.NotZero(t, len(ips))

			v, ok := r.cache.Load(tt.name)
			assert.Equal(t, ips, v)
			assert.True(t, ok)
		}
	})

	t.Run("w/cache", func(t *testing.T) {
		var r Resolver
		tests := []struct {
			name     string
			expected []net.IP
		}{
			{
				"google.com",
				[]net.IP{net.IP("1.1.1.1")},
			},
			{
				"yahoo.co.jp",
				[]net.IP{net.IP("2.2.2.2")},
			},
		}

		for _, tt := range tests {
			r.cache.Store(tt.name, tt.expected)

			ips, err := r.LookupIP(context.Background(), tt.name)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, tt.expected, ips)
		}
	})
}

func TestRefresh(t *testing.T) {
	var r Resolver

	expected := []net.IP{net.IP("1.1.1.1")}
	r.lookupIPFn = func(ctx context.Context, host string) ([]net.IP, error) {
		return expected, nil
	}

	tests := map[string][]net.IP{
		"google.com":  {net.IP("127.0.0.1")},
		"yahoo.co.jp": {net.IP("127.0.0.2")},
	}
	for k, v := range tests {
		r.cache.Store(k, v)
	}

	r.Reflesh()

	for name := range tests {
		ips, err := r.LookupIP(context.Background(), name)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, expected, ips)
	}
}
