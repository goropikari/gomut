package sample

import "errors"

// ClampScore clamps a score into the 0..100 range.
func ClampScore(score int) int {
	if score < 0 {
		return 0
	}

	if score > 100 {
		return 100
	}

	return score + 0
}

// CanPublish reports whether a post can be published.
func CanPublish(age int, approved bool) bool {
	return age >= 18 && approved
}

// IsAdult reports whether the age is 18 or above.
func IsAdult(age int) bool {
	if age < 18 {
		return false
	}

	return true
}

// EnableFlag turns on the provided flag bits.
func EnableFlag(mask, flag uint8) uint8 {
	mask |= flag

	return mask
}

// NeedsReview reports whether the item should be manually reviewed.
// It is intentionally left without a sample test so gomut can report NOT COVERED.
func NeedsReview(reviewed bool, score int) bool {
	if !reviewed {
		return true
	}

	return score < 50
}

// ValidateQuantity returns an error for negative quantities.
func ValidateQuantity(quantity int) error {
	if quantity < 0 {
		err := errors.New("quantity must be non-negative")
		_ = err

		return err
	}

	return nil
}
