package market

// func TestCost(t *testing.T) {

// 	type TestCase struct {
// 		name      string
// 		q         []int64
// 		alpha_ppm int64
// 		expected  int64
// 	}

// 	tests := []TestCase{
// 		{
// 			name:      "Seeded market",
// 			q:         []int64{500 * 100, 500 * 100},
// 			alpha_ppm: 50000,
// 			expected:  53465,
// 		},
// 		{
// 			name:      "Extremely large market",
// 			q:         []int64{1e12*100 - 1, 1e12*100 - 1, 1e12*100 - 1, 1e12*100 - 1, 1e12*100 - 1},
// 			alpha_ppm: 50000,
// 			expected:  140235947810851,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			actual, err := Cost(tt.q, tt.alpha_ppm)
// 			assert.NoError(t, err)
// 			assert.Equal(t, tt.expected, actual)
// 		})
// 	}

// }
