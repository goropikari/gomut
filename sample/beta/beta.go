package beta

// ClampScore clamps a score into the 0..100 range.
func ClampScore(score int) int {
	if score < 0 {
		return 0
	}

	if score > 100 {
		return 100
	}

	return score
}

// CanPublish reports whether a post can be published.
func CanPublish(age int, approved bool) bool {
	return age >= 18 && approved
}
