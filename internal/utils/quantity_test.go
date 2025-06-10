package utils

import (
	"testing"
)

const (
	KB = 1024
	MB = 1024 * KB
	GB = 1024 * MB
	TB = 1024 * GB
	PB = 1024 * TB
	EB = 1024 * PB
)

func TestQuantity_Optimize(t *testing.T) {
	tests := []struct {
		name string
		arg  Quantity
		want Quantity
	}{
		{"t1", Quantity{1, ""}, Quantity{1, ""}},
		{"t2", Quantity{1 * KB, ""}, Quantity{1, "K"}},
		{"t3", Quantity{3 * KB, ""}, Quantity{3, "K"}},
		{"t4", Quantity{1 * MB, ""}, Quantity{1, "M"}},
		{"t5", Quantity{2048, "K"}, Quantity{2, "M"}},
		{"t6", Quantity{2049, "M"}, Quantity{2049, "M"}},
		{"t7", Quantity{3000, "m"}, Quantity{3, ""}},
		{"t8", Quantity{300, "m"}, Quantity{300, "m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := tt.arg.Optimize()
			if res != tt.want {
				t.Errorf("Optimize = %v, want %v", res, tt.want)
			}
		})
	}
}

func TestQuantity_Minus(t *testing.T) {
	tests := []struct {
		name    string
		lhs     Quantity
		rhs     Quantity
		wantErr bool
		want    Quantity
	}{
		{"t1", Quantity{1, ""}, Quantity{0, ""}, false, Quantity{1, ""}},
		{"t2", Quantity{1 * KB, ""}, Quantity{1, ""}, false, Quantity{1023, ""}},
		{"t3", Quantity{3 * KB, ""}, Quantity{0, ""}, false, Quantity{3072, ""}},
		{"t4", Quantity{1 * MB, ""}, Quantity{0, ""}, false, Quantity{1 * MB, ""}},
		{"t5", Quantity{2048, "K"}, Quantity{0, ""}, false, Quantity{2 * MB, ""}},
		{"t6", Quantity{2049, "M"}, Quantity{0, ""}, false, Quantity{2049 * MB, ""}},
		{"t7", Quantity{3000, "m"}, Quantity{0, ""}, false, Quantity{3000, "m"}},
		{"t8", Quantity{300, "m"}, Quantity{0, ""}, false, Quantity{300, "m"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := QuantityMinus(tt.lhs, tt.rhs)
			if (err != nil) != tt.wantErr {
				t.Errorf("QuantityMinus error = %v, wantErr %v", err, tt.wantErr)
			}
			if res != tt.want {
				t.Errorf("QuantityMinus = %v, want %v", res, tt.want)
			}
		})
	}
}
