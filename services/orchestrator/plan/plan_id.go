package plan

import (
	"fmt"
	"math/rand"
	"time"
)

var rng = rand.New(rand.NewSource(time.Now().UnixNano()))

// NewPlanID generates a unique plan identifier.
func NewPlanID() string {
	return fmt.Sprintf("plan_%d_%04d", time.Now().UnixMilli(), rng.Intn(10000))
}
