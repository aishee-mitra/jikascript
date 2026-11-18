package ch32fun

import "testing"

func TestIsValidUARTTuple(t *testing.T) {
	if !IsValidUARTTuple("PD5", "PD6") {
		t.Fatal("expected PD5/PD6 to be a valid uart pair")
	}
	if !IsValidUARTTuple(" pd5 ", "pd6") {
		t.Fatal("expected uart pin names to be normalized before matching")
	}
	if IsValidUARTTuple("PD6", "PD5") {
		t.Fatal("expected reversed uart pair to be invalid")
	}
	if IsValidUARTTuple("PA2", "PA1") {
		t.Fatal("expected non-USART pins to be invalid")
	}
}

func TestIsValidUARTMethod(t *testing.T) {
	for _, method := range []string{"write", "println", "read"} {
		if !IsValidUARTMethod(method) {
			t.Fatalf("expected %s to be a valid uart method", method)
		}
	}
	if IsValidUARTMethod("print") {
		t.Fatal("expected print to be an invalid uart method")
	}
}

func TestIsValidPWMPin(t *testing.T) {
	for _, pin := range []string{"PD2", "PA1", "PC3", "PC4", "PC1", "PD3", "PC0", "PD7"} {
		if !IsValidPWMPin(pin) {
			t.Fatalf("expected %s to be a valid pwm pin", pin)
		}
	}
	if IsValidPWMPin("PD4") {
		t.Fatal("expected PD4 (default TIM2 CH1) to be invalid while TIM2 is remapped")
	}
	if IsValidPWMPin("PD1") {
		t.Fatal("expected PD1 (complementary channel) to be invalid in v1")
	}
}

func TestIsValidPWMMethod(t *testing.T) {
	for _, method := range []string{"set", "setFrequency"} {
		if !IsValidPWMMethod(method) {
			t.Fatalf("expected %s to be a valid pwm method", method)
		}
	}
	if IsValidPWMMethod("pulse") {
		t.Fatal("expected pulse to be an invalid pwm method")
	}
}

func TestIsValidADCPin(t *testing.T) {
	for _, pin := range []string{"PA2", "PA1", "PC4", "PD2", "PD3", "PD5", "PD6", "PD4"} {
		if !IsValidADCPin(pin) {
			t.Fatalf("expected %s to be a valid adc pin", pin)
		}
	}
	if IsValidADCPin("PC1") {
		t.Fatal("expected PC1 to be an invalid adc pin")
	}
}

func TestIsValidADCMethod(t *testing.T) {
	if !IsValidADCMethod("read") {
		t.Fatal("expected read to be a valid adc method")
	}
	if IsValidADCMethod("write") {
		t.Fatal("expected write to be an invalid adc method")
	}
}
