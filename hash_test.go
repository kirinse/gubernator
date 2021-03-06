package gubernator

import (
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/segmentio/fasthash/fnv1"
	"github.com/segmentio/fasthash/fnv1a"
	"github.com/stretchr/testify/assert"
)

func TestConsistantHash(t *testing.T) {
	hosts := []string{"a.svc.local", "b.svc.local", "c.svc.local"}

	t.Run("Add", func(t *testing.T) {
		cases := map[string]string{
			"a":                                    hosts[1],
			"foobar":                               hosts[0],
			"192.168.1.2":                          hosts[1],
			"5f46bb53-6c30-49dc-adb4-b7355058adb6": hosts[1],
		}
		hash := NewConsistantHash(nil)
		for _, h := range hosts {
			hash.Add(&PeerClient{info: PeerInfo{Address: h}})
		}

		for input, addr := range cases {
			t.Run(input, func(t *testing.T) {
				peer, err := hash.Get(input)
				assert.Nil(t, err)
				assert.Equal(t, addr, peer.info.Address)
			})
		}

	})

	t.Run("Size", func(t *testing.T) {
		hash := NewConsistantHash(nil)

		for _, h := range hosts {
			hash.Add(&PeerClient{info: PeerInfo{Address: h}})
		}

		assert.Equal(t, len(hosts), hash.Size())
	})

	t.Run("Host", func(t *testing.T) {
		hash := NewConsistantHash(nil)
		hostMap := map[string]*PeerClient{}

		for _, h := range hosts {
			peer := &PeerClient{info: PeerInfo{Address: h}}
			hash.Add(peer)
			hostMap[h] = peer
		}

		for host, peer := range hostMap {
			assert.Equal(t, peer, hash.GetByPeerInfo(PeerInfo{Address: host}))
		}
	})

	t.Run("distribution", func(t *testing.T) {
		const cases = 10000
		rand.Seed(time.Now().Unix())

		strings := make([]string, cases)

		for i := 0; i < cases; i++ {
			r := rand.Int31()
			ip := net.IPv4(192, byte(r>>16), byte(r>>8), byte(r))
			strings[i] = ip.String()
		}

		hashFuncs := map[string]HashFunc{
			"fasthash/fnv1a": fnv1a.HashBytes32,
			"fasthash/fnv1":  fnv1.HashBytes32,
			"crc32":          nil,
		}

		for name, hashFunc := range hashFuncs {
			t.Run(name, func(t *testing.T) {
				hash := NewConsistantHash(hashFunc)
				hostMap := map[string]int{}

				for _, h := range hosts {
					hash.Add(&PeerClient{info: PeerInfo{Address: h}})
					hostMap[h] = 0
				}

				for i := range strings {
					peer, _ := hash.Get(strings[i])
					hostMap[peer.info.Address]++
				}

				for host, a := range hostMap {
					t.Logf("host: %s, percent: %f", host, float64(a)/cases)
				}
			})
		}
	})

}

func BenchmarkConsistantHash(b *testing.B) {
	hashFuncs := map[string]HashFunc{
		"fasthash/fnv1a": fnv1a.HashBytes32,
		"fasthash/fnv1":  fnv1.HashBytes32,
		"crc32":          nil,
	}

	for name, hashFunc := range hashFuncs {
		b.Run(name, func(b *testing.B) {
			ips := make([]string, b.N)
			for i := range ips {
				ips[i] = net.IPv4(byte(i>>24), byte(i>>16), byte(i>>8), byte(i)).String()
			}

			hash := NewConsistantHash(hashFunc)
			hosts := []string{"a.svc.local", "b.svc.local", "c.svc.local"}
			for _, h := range hosts {
				hash.Add(&PeerClient{info: PeerInfo{Address: h}})
			}

			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				hash.Get(ips[i])
			}
		})
	}
}
