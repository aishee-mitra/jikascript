package ch32fun

// Peripheral capability tables for the CH32V003 target.
//
// The compiler validates every peripheral pin and method against these tables
// before code generation, so the generated C can rely on the target backend
// accepting the same names. Keep the tables in sync with the runtime mapping in
// targets/ch32fun/runtime/*.c.

// uartPairs lists the supported USART1 pin pairs as "TX|RX". The CH32V003 has a
// single USART and its default (no-remap) mapping is TX on PD5 and RX on PD6.
var uartPairs = map[string]struct{}{
	"PD5|PD6": {},
}

// IsValidUARTTuple reports whether tx and rx form a supported UART pin pair.
// Both names are normalized before matching, so case and surrounding
// whitespace do not matter.
func IsValidUARTTuple(tx, rx string) bool {
	_, ok := uartPairs[NormalizePin(tx)+"|"+NormalizePin(rx)]
	return ok
}

// IsValidUARTMethod reports whether a method is valid on a JikaScript uart
// object.
func IsValidUARTMethod(method string) bool {
	switch method {
	case "write", "println", "read":
		return true
	default:
		return false
	}
}

// pwmPins lists the pins that can source a PWM output. The mapping uses the
// default (no-remap) timer alternate functions:
//
//	TIM1 CH1=PD2, CH2=PA1, CH3=PC3, CH4=PC4
//	TIM2 CH1=PD4, CH2=PD3, CH3=PC0, CH4=PD7
//
// Pins that need a timer remap (for example PC1) are intentionally excluded
// from v1 so the backend never depends on AFIO remap state.
var pwmPins = map[string]struct{}{
	"PD2": {}, "PA1": {}, "PC3": {}, "PC4": {},
	"PD4": {}, "PD3": {}, "PC0": {}, "PD7": {},
}

// IsValidPWMPin reports whether the CH32V003 target supports PWM output on the
// supplied pin.
func IsValidPWMPin(name string) bool {
	_, ok := pwmPins[NormalizePin(name)]
	return ok
}

// IsValidPWMMethod reports whether a method is valid on a JikaScript pwm
// object.
func IsValidPWMMethod(method string) bool {
	switch method {
	case "set", "setFrequency":
		return true
	default:
		return false
	}
}

// adcPins lists the pins that can be used as single-ended analog inputs. The
// CH32V003 ADC exposes eight external channels:
//
//	A0=PA2, A1=PA1, A2=PC4, A3=PD2, A4=PD3, A5=PD5, A6=PD6, A7=PD4
var adcPins = map[string]struct{}{
	"PA2": {}, "PA1": {}, "PC4": {}, "PD2": {},
	"PD3": {}, "PD5": {}, "PD6": {}, "PD4": {},
}

// IsValidADCPin reports whether the CH32V003 target supports analog input on
// the supplied pin.
func IsValidADCPin(name string) bool {
	_, ok := adcPins[NormalizePin(name)]
	return ok
}

// IsValidADCMethod reports whether a method is valid on a JikaScript adc
// object.
func IsValidADCMethod(method string) bool {
	return method == "read"
}
