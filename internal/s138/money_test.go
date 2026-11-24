package s138

import "testing"

func TestRound2(t *testing.T) {
	cases := []struct{ in, want float64 }{
		{0, 0},
		// 1.005 has no exact binary representation and is really a shade under
		// 1.005, so it rounds down. Pinned here so the behaviour is known
		// rather than discovered: amounts are the user's own figures and the
		// deadlines do not turn on a paise.
		{1.005, 1.00},
		{2.344, 2.34},
		{2.345, 2.35},
		{-2.345, -2.35},
		{250000, 250000},
		{1.0 / 3.0, 0.33},
	}
	for _, c := range cases {
		if got := Round2(c.in); got != c.want {
			t.Errorf("Round2(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestFormatINRUsesIndianGrouping(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "0.00"},
		{5, "5.00"},
		{99.5, "99.50"},
		{999, "999.00"},
		{1000, "1,000.00"},
		{9999, "9,999.00"},
		{10000, "10,000.00"},
		{99999, "99,999.00"},
		{100000, "1,00,000.00"},
		{250000, "2,50,000.00"},
		{1234567.5, "12,34,567.50"},
		{10000000, "1,00,00,000.00"},
		{12345678, "1,23,45,678.00"},
		{123456789, "12,34,56,789.00"},
		{1234567890, "1,23,45,67,890.00"},
		{-250000.25, "-2,50,000.25"},
		{0.05, "0.05"},
		{0.005, "0.01"},
		{99.999, "100.00"}, // the paise carry into the rupees
	}
	for _, c := range cases {
		if got := FormatINR(c.in); got != c.want {
			t.Errorf("FormatINR(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIndianWords(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "Zero"},
		{1, "One"},
		{9, "Nine"},
		{10, "Ten"},
		{13, "Thirteen"},
		{19, "Nineteen"},
		{20, "Twenty"},
		{21, "Twenty One"},
		{70, "Seventy"},
		{99, "Ninety Nine"},
		{100, "One Hundred"},
		{101, "One Hundred One"},
		{115, "One Hundred Fifteen"},
		{999, "Nine Hundred Ninety Nine"},
		{1000, "One Thousand"},
		{1001, "One Thousand One"},
		{10000, "Ten Thousand"},
		{99999, "Ninety Nine Thousand Nine Hundred Ninety Nine"},
		{100000, "One Lakh"},
		{123456, "One Lakh Twenty Three Thousand Four Hundred Fifty Six"},
		{250000, "Two Lakh Fifty Thousand"},
		{1000000, "Ten Lakh"},
		{10000000, "One Crore"},
		{12345678, "One Crore Twenty Three Lakh Forty Five Thousand Six Hundred Seventy Eight"},
		{100000000, "Ten Crore"},
		{1230000000, "One Hundred Twenty Three Crore"},
		{-500, "Minus Five Hundred"},
	}
	for _, c := range cases {
		if got := IndianWords(c.in); got != c.want {
			t.Errorf("IndianWords(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestAmountInWordsMatchesTheFigure matters because a demand notice states the
// amount twice, in figures and in words, and the two have to agree.
func TestAmountInWordsMatchesTheFigure(t *testing.T) {
	cases := []struct {
		in   float64
		want string
	}{
		{0, "Rupees Zero only"},
		{1, "Rupees One only"},
		{250000, "Rupees Two Lakh Fifty Thousand only"},
		{250000.75, "Rupees Two Lakh Fifty Thousand and Paise Seventy Five only"},
		{1234567.05, "Rupees Twelve Lakh Thirty Four Thousand Five Hundred Sixty Seven and Paise Five only"},
		{99.999, "Rupees One Hundred only"}, // the carry must reach the words too
		{-500.50, "Minus Rupees Five Hundred and Paise Fifty only"},
	}
	for _, c := range cases {
		if got := AmountInWords(c.in); got != c.want {
			t.Errorf("AmountInWords(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}
