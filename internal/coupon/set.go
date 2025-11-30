package coupon

// mapCouponSet implements CouponSet using a map for O(1) lookups.
type mapCouponSet struct {
	coupons map[string]struct{}
}

// NewMapCouponSet creates a new map-based coupon set.
func NewMapCouponSet(capacity int) CouponSet {
	return &mapCouponSet{
		coupons: make(map[string]struct{}, capacity),
	}
}

// Contains checks if a coupon code exists in the set.
func (s *mapCouponSet) Contains(code string) bool {
	_, exists := s.coupons[code]
	return exists
}

// Size returns the number of coupons in the set.
func (s *mapCouponSet) Size() int {
	return len(s.coupons)
}

// Add adds a coupon code to the set.
func (s *mapCouponSet) Add(code string) {
	s.coupons[code] = struct{}{}
}
