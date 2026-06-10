package pruning

// MessageWithID is a message that carries an ID for pin matching.
type MessageWithID struct {
	ID   string
	Role string
	Text string
}

// PreparePinnedMessages filters messages using a head+tail window plus pinned
// overrides. Pinned message IDs that are not found are returned as missingIDs
// so the caller can log a warning; they are not treated as errors.
//
// head and tail are the number of messages to always keep from each end.
// When both are 0 and pinnedIDs is empty, all messages are returned unfiltered.
func PreparePinnedMessages(messages []MessageWithID, pinnedIDs []string, head, tail int) (filtered []MessageWithID, missingIDs []string) {
	if len(messages) == 0 {
		if len(pinnedIDs) > 0 {
			return nil, append([]string(nil), pinnedIDs...)
		}
		return nil, nil
	}

	// Build pinned set from IDs, mapping ID → whether it was found.
	pinnedSet := make(map[string]bool, len(pinnedIDs))
	for _, id := range pinnedIDs {
		if id != "" {
			pinnedSet[id] = false
		}
	}

	// Map message ID → index.
	idToIndex := make(map[string]int, len(messages))
	for i, m := range messages {
		if m.ID != "" {
			idToIndex[m.ID] = i
		}
	}

	// Mark found pinned IDs.
	var missing []string
	for id := range pinnedSet {
		if _, ok := idToIndex[id]; ok {
			pinnedSet[id] = true
		} else {
			missing = append(missing, id)
		}
	}

	total := len(messages)

	// When no window is specified, return all messages (pins only add overrides,
	// and unknown pins are reported but do not drop messages).
	if head == 0 && tail == 0 {
		return append([]MessageWithID(nil), messages...), missing
	}

	// Clamp head/tail to total.
	if head > total {
		head = total
	}
	if head < 0 {
		head = 0
	}
	if tail > total {
		tail = total
	}
	if tail < 0 {
		tail = 0
	}

	keep := make([]bool, total)

	// Head window.
	for i := 0; i < head; i++ {
		keep[i] = true
	}

	// Tail window.
	for i := total - tail; i < total; i++ {
		keep[i] = true
	}

	// Pinned overrides.
	for id, found := range pinnedSet {
		if !found {
			continue
		}
		if idx, ok := idToIndex[id]; ok {
			keep[idx] = true
		}
	}

	filtered = make([]MessageWithID, 0, total)
	for i, m := range messages {
		if keep[i] {
			filtered = append(filtered, m)
		}
	}

	return filtered, missing
}
