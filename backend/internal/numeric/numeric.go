package numeric

import (
	"fmt"

	"github.com/ericlagergren/decimal"
	"github.com/jackc/pgx/v5/pgtype"
)

const Scale = 12

type BigDecimal struct {
	decimal.Big
}

func (d *BigDecimal) ScanNumeric(v pgtype.Numeric) error {

	if !v.Valid {
		return fmt.Errorf("cannot scan NULL into *BigDecimal")
	}

	if v.NaN {
		return fmt.Errorf("cannot scan NaN into *BigDecimal")
	}

	if v.InfinityModifier != pgtype.Finite {
		return fmt.Errorf("cannot scan %v into *BigDecimal", v.InfinityModifier)
	}

	d.SetBigMantScale(v.Int, -int(v.Exp))

	return nil
}

func (d BigDecimal) NumericValue() (pgtype.Numeric, error) {
	var n pgtype.Numeric
	if d.Sign() == 0 {
		if err := n.Scan("0"); err != nil {
			return n, err
		}
		return n, nil
	}
	return n, n.Scan(d.String())
}
