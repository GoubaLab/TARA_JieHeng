package assurance

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
)

// decimalRat uses the finite JSON decimal spelling, not the binary floating-point
// expansion. All arithmetic and threshold comparisons remain exact until output.
func decimalRat(value any) *big.Rat {
	if v, ok := value.(*big.Rat); ok {
		return new(big.Rat).Set(v)
	}
	var text string
	switch v := value.(type) {
	case json.Number:
		text = string(v)
	case float64:
		text = strconv.FormatFloat(v, 'g', -1, 64)
	case float32:
		text = strconv.FormatFloat(float64(v), 'g', -1, 32)
	default:
		text = fmt.Sprint(v)
	}
	result, ok := new(big.Rat).SetString(text)
	if !ok {
		return new(big.Rat)
	}
	return result
}

func decimalNumber(value *big.Rat) float64 { result, _ := value.Float64(); return result }

func coefficientTotal(method Object) *big.Rat {
	total := new(big.Rat)
	for _, key := range impactKeys {
		total.Add(total, decimalRat(dObj(method["impact_coefficients"])[key]))
	}
	return total
}

func coefficientImpact(impact, method Object) (*big.Rat, bool) {
	total := new(big.Rat)
	coefficients := dObj(method["impact_coefficients"])
	for _, key := range impactKeys {
		coefficient := decimalRat(coefficients[key])
		if coefficient.Sign() == 0 {
			continue
		}
		if impact[key] == nil {
			return nil, false
		}
		total.Add(total, new(big.Rat).Mul(decimalRat(impact[key]), coefficient))
	}
	return total, coefficientTotal(method).Sign() > 0
}

func coefficientRisk(impact any, feasibility int, method Object) (int, float64) {
	score := new(big.Rat).Mul(decimalRat(impact), big.NewRat(int64(feasibility), 1))
	for i, threshold := range dList(method["product_thresholds"]) {
		if score.Cmp(decimalRat(threshold)) <= 0 {
			return i + 1, decimalNumber(score)
		}
	}
	return 5, decimalNumber(score)
}

func validateResidualImpact(value any, method Object) (any, error) {
	if method["impact_aggregation"] != "coefficient_sum" {
		return dValidateField(dn("int", 0, 3, true), value)
	}
	maximum := new(big.Rat).Mul(coefficientTotal(method), big.NewRat(3, 1))
	if decimalRat(value).Cmp(maximum) > 0 {
		return nil, dError("残余影响总分不得超过各维最高等级与当前系数计算的上限")
	}
	return value, nil
}
