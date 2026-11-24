package s138

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Round2 rounds to paise. Half-away-from-zero, which is how a figure is
// rounded on a notice.
func Round2(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return math.Round(v*100) / 100
}

// FormatINR renders an amount with Indian digit grouping and two decimals:
// 1234567.5 becomes "12,34,567.50". The last three digits form one group and
// everything before it is grouped in twos.
func FormatINR(v float64) string {
	v = Round2(v)
	sign := ""
	if v < 0 {
		sign, v = "-", -v
	}
	whole := int64(v)
	paise := int64(math.Round((v - float64(whole)) * 100))
	if paise == 100 { // the rounding above can carry
		whole++
		paise = 0
	}
	return fmt.Sprintf("%s%s.%02d", sign, groupIndian(whole), paise)
}

// groupIndian puts commas into a non-negative integer the Indian way.
func groupIndian(n int64) string {
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		return s
	}
	head, tail := s[:len(s)-3], s[len(s)-3:]
	var parts []string
	for len(head) > 2 {
		parts = append([]string{head[len(head)-2:]}, parts...)
		head = head[:len(head)-2]
	}
	if head != "" {
		parts = append([]string{head}, parts...)
	}
	return strings.Join(parts, ",") + "," + tail
}

var onesWords = [20]string{
	"", "One", "Two", "Three", "Four", "Five", "Six", "Seven", "Eight", "Nine",
	"Ten", "Eleven", "Twelve", "Thirteen", "Fourteen", "Fifteen", "Sixteen",
	"Seventeen", "Eighteen", "Nineteen",
}

var tensWords = [10]string{
	"", "", "Twenty", "Thirty", "Forty", "Fifty", "Sixty", "Seventy", "Eighty", "Ninety",
}

// underHundred spells 0..99. Zero spells as the empty string: the callers add
// "Zero" only when the whole number is zero.
func underHundred(n int64) string {
	switch {
	case n < 20:
		return onesWords[n]
	case n%10 == 0:
		return tensWords[n/10]
	default:
		return tensWords[n/10] + " " + onesWords[n%10]
	}
}

// IndianWords spells a non-negative whole number in the Indian system, using
// crore and lakh. It is used for the amount in words on the demand notice,
// where a figure and its words have to agree.
func IndianWords(n int64) string {
	if n < 0 {
		return "Minus " + IndianWords(-n)
	}
	if n == 0 {
		return "Zero"
	}
	var parts []string
	if crore := n / 10000000; crore > 0 {
		parts = append(parts, IndianWords(crore)+" Crore")
		n %= 10000000
	}
	if lakh := n / 100000; lakh > 0 {
		parts = append(parts, underHundred(lakh)+" Lakh")
		n %= 100000
	}
	if thousand := n / 1000; thousand > 0 {
		parts = append(parts, underHundred(thousand)+" Thousand")
		n %= 1000
	}
	if hundred := n / 100; hundred > 0 {
		parts = append(parts, onesWords[hundred]+" Hundred")
		n %= 100
	}
	if n > 0 {
		parts = append(parts, underHundred(n))
	}
	return strings.Join(parts, " ")
}

// AmountInWords renders a rupee amount the way it is written on a notice:
// "Rupees One Lakh Twenty Three Thousand Four Hundred Fifty Six and Paise
// Seventy Eight only".
func AmountInWords(v float64) string {
	v = Round2(v)
	sign := ""
	if v < 0 {
		sign, v = "Minus ", -v
	}
	whole := int64(v)
	paise := int64(math.Round((v - float64(whole)) * 100))
	if paise == 100 {
		whole++
		paise = 0
	}
	out := sign + "Rupees " + IndianWords(whole)
	if paise > 0 {
		out += " and Paise " + underHundred(paise)
	}
	return out + " only"
}
