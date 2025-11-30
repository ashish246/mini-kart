package coupon

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMapCouponSet_Add_And_Contains(t *testing.T) {
	set := NewMapCouponSet(10).(*mapCouponSet)

	// Test adding and checking presence
	set.Add("TESTCODE1")
	assert.True(t, set.Contains("TESTCODE1"))
	assert.False(t, set.Contains("NOTEXIST"))

	// Test multiple additions
	set.Add("TESTCODE2")
	set.Add("TESTCODE3")
	assert.True(t, set.Contains("TESTCODE1"))
	assert.True(t, set.Contains("TESTCODE2"))
	assert.True(t, set.Contains("TESTCODE3"))

	// Test duplicate addition (should not increase size)
	set.Add("TESTCODE1")
	assert.Equal(t, 3, set.Size())
}

func TestMapCouponSet_Size(t *testing.T) {
	tests := []struct {
		name     string
		codes    []string
		expected int
	}{
		{
			name:     "Empty set",
			codes:    []string{},
			expected: 0,
		},
		{
			name:     "Single code",
			codes:    []string{"CODE123"},
			expected: 1,
		},
		{
			name:     "Multiple unique codes",
			codes:    []string{"CODE1", "CODE2", "CODE3"},
			expected: 3,
		},
		{
			name:     "Duplicate codes",
			codes:    []string{"CODE1", "CODE1", "CODE2"},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set := NewMapCouponSet(10).(*mapCouponSet)

			for _, code := range tt.codes {
				set.Add(code)
			}

			assert.Equal(t, tt.expected, set.Size())
		})
	}
}

func TestMapCouponSet_Contains(t *testing.T) {
	set := NewMapCouponSet(10).(*mapCouponSet)
	set.Add("VALIDCODE")
	set.Add("TESTPROMO")
	set.Add("DISCOUNT10")

	tests := []struct {
		name     string
		code     string
		expected bool
	}{
		{
			name:     "Code exists",
			code:     "VALIDCODE",
			expected: true,
		},
		{
			name:     "Code does not exist",
			code:     "INVALID",
			expected: false,
		},
		{
			name:     "Empty string",
			code:     "",
			expected: false,
		},
		{
			name:     "Case sensitive - exact match",
			code:     "TESTPROMO",
			expected: true,
		},
		{
			name:     "Case sensitive - different case",
			code:     "testpromo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := set.Contains(tt.code)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapCouponSet_Capacity(t *testing.T) {
	// Test that pre-allocation works without errors
	largeCap := 1_000_000
	set := NewMapCouponSet(largeCap).(*mapCouponSet)

	assert.NotNil(t, set.coupons)
	assert.Equal(t, 0, set.Size())

	// Add some codes
	for i := 0; i < 100; i++ {
		set.Add("CODE" + string(rune(i)))
	}

	assert.Equal(t, 100, set.Size())
}
