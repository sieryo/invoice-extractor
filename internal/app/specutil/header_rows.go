package specutil

func HeaderRowNumbers(max int) []int {
	if max < 1 {
		max = 1
	}

	rows := make([]int, 0, max)
	for i := 1; i <= max; i++ {
		rows = append(rows, i)
	}
	return rows
}
