package toml

import (
	"fmt"
	"unicode"

	"github.com/pkg/errors"
)

type from func(rune) (int, error)

func fromHex(r rune) (int, error) {

	if unicode.IsOneOf([]*unicode.RangeTable{digits}, r) {

		return int(r) - 48, nil
	}

	if unicode.IsOneOf([]*unicode.RangeTable{hex}, r) {

		v := int(r)

		if v > 70 {
			return v - 97 + 10, nil
		}

		return v - 65 + 10, nil
	}

	return 0, fmt.Errorf("not a valid hex value")
}

func fromOctal(r rune) (int, error) {

	v := int(r)

	if v >= 48 || v <= 55 {
		return v - 48, nil
	}
	return 0, fmt.Errorf(`not valid octal value`)
}

func fromBin(r rune) (int, error) {

	if r == '1' {
		return 1, nil
	}

	if r == '0' {
		return 0, nil
	}

	return 0, fmt.Errorf(`not valid bin value`)
}

func addNumber(r rune, total int64, t NumberType) (int64, error) {

	switch t {
	case BinNumberType:
		return add(r, total, fromBin, 2)
	case OctalNumberType:
		return add(r, total, fromOctal, 8)
	case HexNumberType:
		return add(r, total, fromHex, 16)
	}

	return 0, fmt.Errorf(`addRune type not supported`)
}

func add(r rune, total int64, f from, base int) (int64, error) {

	v, err := f(r)
	if err != nil {
		return 0, errors.Wrap(err, `add failed`)
	}

	if v == 0 && total == 0 {
		return 0, nil
	}

	total *= int64(base)
	total += int64(v)
	return total, nil
}
