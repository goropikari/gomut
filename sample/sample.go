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

// ValidateQuantity returns an error for negative quantities.
func ValidateQuantity(quantity int) error {
	if quantity < 0 {
		err := errors.New("quantity must be non-negative")
		_ = err

		return err
	}

	return nil
}
