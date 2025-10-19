package market

// func TestSoftmax(t *testing.T) {

// 	type TestCase struct {
// 		name     string
// 		q        []int64
// 		alphaPPM int64
// 		expected []string // softmax precise to 20 digits, rounded down
// 		// We multipy the result by 10**20, and floor it
// 	}

// 	tests := []TestCase{
// 		{
// 			name:     "Pathological test case",
// 			q:        []int64{543_792_481 * 100, 1_234_875_552 * 100, 5_234_753_083 * 100, 43_879_118 * 100},
// 			alphaPPM: 50_000,
// 			expected: []string{"168467497565065", "1194202863984192", "99998596475181799681", "40854456651060"},
// 		},
// 		{
// 			name:     "Two equal softmaxes",
// 			q:        []int64{500 * 100, 500 * 100},
// 			alphaPPM: 50_000,
// 			expected: []string{"50000000000000000000", "50000000000000000000"},
// 		},
// 		{
// 			name:     "Extremely large equal market",
// 			q:        []int64{1_000_000_000_000 * 100, 1_000_000_000_000 * 100, 1_000_000_000_000 * 100},
// 			alphaPPM: 50_000,
// 			// 1/3 * 1e20 floored, last digit is 3
// 			expected: []string{"33333333333333333333", "33333333333333333333", "33333333333333333333"},
// 		},
// 		{
// 			name:     "Magnitude difference",
// 			q:        []int64{37_518_378_724 * 100, 1_234_311_111 * 100, 100 * 100, 123_123_123_444_322 * 100, 5, 1},
// 			alphaPPM: 50_000,
// 			expected: []string{"208684076403", "207458105522", "207416527369", "99999998961608235970", "207416527366", "207416527366"},
// 		},
// 		{
// 			name:     "High alpha",
// 			q:        []int64{37_518_378_724 * 100, 1_234_311_111 * 100, 100 * 100, 123_123_123_444_322 * 100, 5, 1},
// 			alphaPPM: 123_456,
// 			expected: []string{"30458608855667965", "30386011725079462", "30383545162267491", "99848004743932849626", "30383545162067766", "30383545162067686"},
// 		},
// 	}

// 	var pw10_20 decimal.Big
// 	ctx.Pow(&pw10_20, decimal.New(10, 0), decimal.New(20, 0))
// 	assert.NoError(t, ctx.Err())

// 	for _, tt := range tests {

// 		t.Run(tt.name, func(t *testing.T) {

// 			b, err := ComputeBDec(tt.q, tt.alphaPPM)
// 			assert.NoError(t, err)

// 			s, err := SoftmaxB(tt.q, b)
// 			assert.NoError(t, err)

// 			for i, si := range s {
// 				var x decimal.Big

// 				ctx.Mul(&x, si, &pw10_20)
// 				assert.NoError(t, ctx.Err())

// 				var xRound decimal.Big
// 				ctx.Floor(&xRound, &x)
// 				assert.NoError(t, ctx.Err())

// 				decimalsStr := xRound.String()
// 				assert.Equal(t, tt.expected[i], decimalsStr)

// 			}

// 		})
// 	}

// }
