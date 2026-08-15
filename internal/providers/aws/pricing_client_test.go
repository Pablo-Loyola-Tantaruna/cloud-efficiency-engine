package aws

import (
	"testing"
)

func TestParseOnDemandHourlyPrice(
	t *testing.T,
) {

	raw :=
		`{
			"terms": {
				"OnDemand": {
					"TEST": {
						"priceDimensions": {
							"PRICE": {
								"unit": "Hrs",
								"beginRange": "0",
								"pricePerUnit": {
									"USD": "0.0960000000"
								}
							}
						}
					}
				}
			}
		}`

	price, err :=
		parseOnDemandHourlyPrice(
			raw,
		)

	if err != nil {
		t.Fatalf(
			"expected no error, got %v",
			err,
		)
	}

	if price != 0.096 {
		t.Fatalf(
			"expected 0.096, got %f",
			price,
		)
	}
}

func TestParseOnDemandHourlyPrice_ShouldIgnoreNonHourly(
	t *testing.T,
) {

	raw :=
		`{
			"terms": {
				"OnDemand": {
					"TEST": {
						"priceDimensions": {
							"PRICE": {
								"unit": "GB-Mo",
								"beginRange": "0",
								"pricePerUnit": {
									"USD": "0.0960000000"
								}
							}
						}
					}
				}
			}
		}`

	_, err :=
		parseOnDemandHourlyPrice(
			raw,
		)

	if err == nil {
		t.Fatal(
			"expected error",
		)
	}
}

func TestParseOnDemandHourlyPrice_ShouldRejectInvalidJSON(
	t *testing.T,
) {

	_, err :=
		parseOnDemandHourlyPrice(
			`invalid`,
		)

	if err == nil {
		t.Fatal(
			"expected error",
		)
	}
}
