package chariot

// V2Solution is defined once for the whole package and reused by all OS builds.
type V2Solution struct {
	NumItems  int
	Select    []int   // length == NumItems; 0/1 per item
	Objective float64 // sum of weighted objective terms
	Penalty   float64 // total penalty from soft constraints
	Total     float64 // Objective - Penalty
}
