package nearest

import (
	"math"
	"math/big"
	"pfi/sensorbee/sensorbee/data"
	"sort"
)

type Minhash struct {
	bitNum int
	data   []*big.Int
	ndata  int
}

func NewMinhash(bitNum int) *Minhash {
	return &Minhash{
		bitNum: bitNum,
	}
}

func (m *Minhash) SetRow(id ID, v FeatureVector) {
	m.ndata = maxInt(m.ndata, int(id))
	if len(m.data) < int(id) {
		m.extend(int(id))
	}
	m.data[id-1] = m.hash(v)
}

func (m *Minhash) NeighborRowFromID(id ID, size int) []IDist {
	return m.neighborRowFromHash(m.data[id-1], size)
}

func (m *Minhash) NeighborRowFromFV(v FeatureVector, size int) []IDist {
	return m.neighborRowFromHash(m.hash(v), size)
}

func (m *Minhash) neighborRowFromHash(x *big.Int, size int) []IDist {
	return m.rankingHammingBitVectors(x, size)
}

func (m *Minhash) GetAllRows() []ID {
	// TODO: implement
	return nil
}

func (m *Minhash) extend(n int) {
	len := maxInt(2*len(m.data), n)
	data := make([]*big.Int, len)
	copy(data, m.data)
	m.data = data
}

func (m *Minhash) rankingHammingBitVectors(bv *big.Int, size int) []IDist {
	ret := make([]IDist, m.ndata)
	for i := 0; i < m.ndata; i++ {
		dist := m.calcHammingDistance(bv, m.data[i])
		ret[i] = IDist{
			ID:   ID(i + 1),
			Dist: float32(dist),
		}
	}
	sort.Sort(sortByDist(ret))
	ret = ret[:minInt(size, len(ret))]
	for i := range ret {
		ret[i].Dist /= float32(m.bitNum)
	}
	return ret
}

type sortByDist []IDist

func (s sortByDist) Len() int {
	return len(s)
}

func (s sortByDist) Less(i, j int) bool {
	return s[i].Dist < s[j].Dist
}

func (s sortByDist) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (m *Minhash) calcHammingDistance(x, y *big.Int) uint64 {
	xb := x.Bits()
	yb := y.Bits()

	minLen := len(yb)
	maxLen := len(xb)
	if len(xb) < len(yb) {
		xb, yb = yb, xb
		minLen, maxLen = maxLen, minLen
	}

	var ret uint64
	for i := 0; i < minLen; i++ {
		ret += bitcount(xb[i] ^ yb[i])
	}
	for i := minLen; i < maxLen; i++ {
		ret += bitcount(xb[i])
	}

	return ret
}

func (m *Minhash) hash(v FeatureVector) *big.Int {
	minValues := generateMinValuesBuffer(m.bitNum)
	hashes := make([]uint64, m.bitNum)
	for dim, val := range v {
		x64, err := data.ToFloat(val)
		if err != nil {
			// TODO: remove panic
			panic(err)
		}
		x := float32(x64)
		keyHash := calcStringHash(dim)
		for i := 0; i < m.bitNum; i++ {
			hashVal := calcHash(keyHash, uint64(i), x)
			if hashVal < minValues[i] {
				minValues[i] = hashVal
				hashes[i] = keyHash
			}
		}
	}

	bv := big.NewInt(0)
	bit := big.NewInt(1)
	for i := 0; i < len(hashes); i++ {
		if (hashes[i] & 1) == 1 {
			bv.Or(bv, bit)
		}
		bit.Lsh(bit, 1)
	}
	return bv
}

func generateMinValuesBuffer(n int) []float32 {
	ret := make([]float32, n)
	for i := 0; i < n; i++ {
		ret[i] = inf32
	}
	return ret
}

var inf32 = float32(math.Inf(1))

func hashMix64(a, b, c uint64) (uint64, uint64, uint64) {
	a -= b
	a -= c
	a ^= (c >> 43)

	b -= c
	b -= a
	b ^= (a << 9)

	c -= a
	c -= b
	c ^= (b >> 8)

	a -= b
	a -= c
	a ^= (c >> 38)

	b -= c
	b -= a
	b ^= (a << 23)

	c -= a
	c -= b
	c ^= (b >> 5)

	a -= b
	a -= c
	a ^= (c >> 35)

	b -= c
	b -= a
	b ^= (a << 49)

	c -= a
	c -= b
	c ^= (b >> 11)

	a -= b
	a -= c
	a ^= (c >> 12)

	b -= c
	b -= a
	b ^= (a << 18)

	c -= a
	c -= b
	c ^= (b >> 22)

	return a, b, c
}

func calcHash(a, b uint64, val float32) float32 {
	const hashPrime = 0xc3a5c85c97cb3127

	var c uint64 = hashPrime
	a, b, c = hashMix64(a, b, c)
	a, b, c = hashMix64(a, b, c)
	r := float32(a) / float32(0xFFFFFFFFFFFFFFFF)
	return -log32(r) / val
}

func calcStringHash(s string) uint64 {
	// FNV-1
	var hash uint64 = 14695981039346656037
	for i := 0; i < len(s); i++ {
		hash *= 1099511628211
		hash ^= uint64(s[i])
	}
	return hash
}

func log32(x float32) float32 {
	return float32(math.Log(float64(x)))
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x < y {
		return y
	}
	return x
}

func bitcount(x big.Word) uint64 {
	var ret uint64
	for x != 0 {
		ret++
		x &= x - 1
	}
	return ret
}
