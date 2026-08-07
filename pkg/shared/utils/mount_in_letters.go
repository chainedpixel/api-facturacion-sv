package utils

import (
	"errors"
	"fmt"
	"math"
	"strings"
)

var (
	ErrValorNoAdmitido = errors.New("value not allowed to be converted to words")
	us                 = []string{"cero", "uno", "dos", "tres", "cuatro", "cinco", "seis", "siete", "ocho", "nueve"}
	ds                 = []string{"X", "y", "veinte", "treinta", "cuarenta", "cincuenta", "sesenta", "setenta", "ochenta", "noventa"}
	des                = []string{"diez", "once", "doce", "trece", "catorce", "quince", "dieciseis", "diecisiete", "dieciocho", "diecinueve"}
	cs                 = []string{"x", "cien", "doscientos", "trescientos", "cuatrocientos", "quinientos", "seiscientos", "setecientos", "ochocientos", "novecientos"}
)

// InLetters converts a number to words in Spanish and returns it in uppercase.
func InLetters(n float64) string {
	if math.IsNaN(n) || math.IsInf(n, 0) || math.Abs(n) >= 1000000000 {
		return ErrValorNoAdmitido.Error()
	}

	message, err := generateMessage(n)
	if err != nil {
		return err.Error()
	}
	return message
}

func generateMessage(n float64) (string, error) {
	var sb strings.Builder

	if n < 0 {
		sb.WriteString("menos ")
		n = -n
	}

	n = math.Round(n*100) / 100
	entera := int(n)
	decimal := int(math.Round((n - float64(entera)) * 100))

	parteEntera, err := convertIntegerToWords(entera)
	if err != nil {
		return "", err
	}
	sb.WriteString(parteEntera)

	if decimal >= 100 {
		nuevaParteEntera, err := convertIntegerToWords(entera + 1)
		if err != nil {
			return "", err
		}
		return strings.ToUpper(nuevaParteEntera + " 00/100"), nil
	}

	sb.WriteString(fmt.Sprintf(" %02d/100", decimal))
	return strings.ToUpper(strings.TrimSpace(sb.String())), nil
}

func handleVeintiNumbers(n int, beforeMil bool) string {
	unidades := n % 10
	if unidades == 0 {
		return "veinte"
	}
	if unidades == 1 {
		if beforeMil {
			return "veintiun"
		}
		return "veintiuno"
	}
	return "veinti" + us[unidades]
}

func convertIntegerToWords(n int) (string, error) {
	if n == 0 {
		return us[0], nil
	}

	var sb strings.Builder

	millones := n / 1000000
	n = n % 1000000
	if millones > 0 {
		if millones == 1 {
			sb.WriteString("un millón ")
		} else {
			millonesStr, _ := convertIntegerToWords(millones)
			sb.WriteString(millonesStr + " millones ")
		}
	}

	miles := n / 1000
	n = n % 1000
	if miles > 0 {
		if miles == 1 {
			sb.WriteString("un mil ")
		} else {
			if miles == 21 {
				sb.WriteString("veintiun mil ")
			} else {
				milesStr, _ := convertIntegerToWords(miles)
				sb.WriteString(milesStr + " mil ")
			}
		}
	}

	centenas := n / 100
	n = n % 100
	if centenas > 0 {
		if centenas == 1 {
			if n == 0 {
				sb.WriteString("cien")
			} else {
				sb.WriteString("ciento ")
			}
		} else {
			sb.WriteString(cs[centenas] + " ")
		}
	}

	if n > 0 {
		if n < 10 {
			sb.WriteString(us[n])
		} else if n < 20 {
			sb.WriteString(des[n-10])
		} else if n < 30 {
			sb.WriteString(handleVeintiNumbers(n, false))
		} else {
			decenas := n / 10
			unidades := n % 10
			sb.WriteString(ds[decenas])
			if unidades > 0 {
				sb.WriteString(" y " + us[unidades])
			}
		}
	}

	return strings.TrimSpace(sb.String()), nil
}
