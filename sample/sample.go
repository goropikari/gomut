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

// IsAllowed reports whether access is allowed.
func IsAllowed(approved bool) bool {
	if approved {
		return true
	}

	return false
}

// HasNickname reports whether the nickname pointer is set.
func HasNickname(nickname *string) bool {
	return nickname != nil
}

// AlwaysEnabled returns a fixed boolean literal.
func AlwaysEnabled() bool {
	return true
}

// AlwaysDisabled returns a fixed boolean literal.
func AlwaysDisabled() bool {
	return false
}

// EnableFlag turns on the provided flag bits.
func EnableFlag(mask, flag uint8) uint8 {
	mask |= flag

	return mask
}

// KeepCommonBits keeps only the bits shared by mask and flag.
func KeepCommonBits(mask, flag uint8) uint8 {
	return mask & flag
}

// MergeFlags combines the provided flags.
func MergeFlags(mask, flag uint8) uint8 {
	return mask | flag
}

// ClearFlagBits removes the provided bits from mask.
func ClearFlagBits(mask, flag uint8) uint8 {
	return mask &^ flag
}

// ShiftLeft shifts the value left by one bit.
func ShiftLeft(value uint8) uint8 {
	return value << 1
}

// ShiftRight shifts the value right by one bit.
func ShiftRight(value uint8) uint8 {
	return value >> 1
}

// ShiftCounter increments the value in place using a shift assignment.
func ShiftCounter(value uint8) uint8 {
	value <<= 1

	return value
}

// NegateScore returns the negated score.
func NegateScore(score int) int {
	return -score
}

// InvertBits returns the bitwise complement of the value.
func InvertBits(value uint8) uint8 {
	return ^value
}

// AddBonus adds a bonus score onto the current score.
func AddBonus(score, bonus int) int {
	score += bonus

	return score
}

// AdvanceCount increments the counter by one.
func AdvanceCount(count int) int {
	count++

	return count
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

// DefaultRetryLimit returns the default retry limit.
func DefaultRetryLimit() int {
	return 3
}

// Greeting returns a fixed greeting.
func Greeting() string {
	return "hello"
}

// DefaultTaxRate returns the default tax rate.
func DefaultTaxRate() float64 {
	return 0.1
}

// DefaultGrade returns the default grade rune.
func DefaultGrade() rune {
	return 'A'
}

// IsBlocked reports whether approval is blocked.
func IsBlocked(approved bool) bool {
	return !approved
}

// ApprovalLabel returns a label for the approval state.
func ApprovalLabel(approved bool) string {
	switch approved {
	case true:
		return "approved"
	default:
		return "blocked"
	}
}

// IgnoredThreshold reports whether a score is within the ignored sample range.
//
//gomut:ignore
func IgnoredThreshold(score int) bool {
	if score < 50 {
		return true
	}

	return false
}

type SomeDefinedType int

const (
	foo SomeDefinedType = 10
	bar SomeDefinedType = 11
)

func ConstVariable() bool {
	return foo < bar
}
