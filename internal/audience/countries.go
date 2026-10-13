package audience

import "strings"

// countryNames maps ISO 3166-1 alpha-2 codes to English short names, for
// the contact profile ("United States · America/Chicago").
var countryNames = map[string]string{}

func init() {
	for _, line := range strings.Split(countryTable, "\n") {
		code, name, ok := strings.Cut(strings.TrimSpace(line), " ")
		if ok {
			countryNames[code] = name
		}
	}
}

// CountryName returns the name of a country code, or the code itself.
func CountryName(code string) string {
	if n, ok := countryNames[strings.ToUpper(strings.TrimSpace(code))]; ok {
		return n
	}
	return code
}

const countryTable = `AD Andorra
AE United Arab Emirates
AF Afghanistan
AG Antigua and Barbuda
AL Albania
AM Armenia
AO Angola
AR Argentina
AT Austria
AU Australia
AZ Azerbaijan
BA Bosnia and Herzegovina
BB Barbados
BD Bangladesh
BE Belgium
BF Burkina Faso
BG Bulgaria
BH Bahrain
BJ Benin
BN Brunei
BO Bolivia
BR Brazil
BS Bahamas
BT Bhutan
BW Botswana
BY Belarus
BZ Belize
CA Canada
CD DR Congo
CH Switzerland
CI Côte d'Ivoire
CL Chile
CM Cameroon
CN China
CO Colombia
CR Costa Rica
CU Cuba
CV Cape Verde
CY Cyprus
CZ Czechia
DE Germany
DK Denmark
DO Dominican Republic
DZ Algeria
EC Ecuador
EE Estonia
EG Egypt
ES Spain
ET Ethiopia
FI Finland
FJ Fiji
FR France
GA Gabon
GB United Kingdom
GE Georgia
GH Ghana
GR Greece
GT Guatemala
HK Hong Kong
HN Honduras
HR Croatia
HT Haiti
HU Hungary
ID Indonesia
IE Ireland
IL Israel
IN India
IQ Iraq
IR Iran
IS Iceland
IT Italy
JM Jamaica
JO Jordan
JP Japan
KE Kenya
KG Kyrgyzstan
KH Cambodia
KR South Korea
KW Kuwait
KZ Kazakhstan
LA Laos
LB Lebanon
LI Liechtenstein
LK Sri Lanka
LT Lithuania
LU Luxembourg
LV Latvia
LY Libya
MA Morocco
MC Monaco
MD Moldova
ME Montenegro
MG Madagascar
MK North Macedonia
ML Mali
MM Myanmar
MN Mongolia
MT Malta
MU Mauritius
MV Maldives
MW Malawi
MX Mexico
MY Malaysia
MZ Mozambique
NA Namibia
NE Niger
NG Nigeria
NI Nicaragua
NL Netherlands
NO Norway
NP Nepal
NZ New Zealand
OM Oman
PA Panama
PE Peru
PG Papua New Guinea
PH Philippines
PK Pakistan
PL Poland
PR Puerto Rico
PT Portugal
PY Paraguay
QA Qatar
RO Romania
RS Serbia
RU Russia
RW Rwanda
SA Saudi Arabia
SE Sweden
SG Singapore
SI Slovenia
SK Slovakia
SN Senegal
SO Somalia
SV El Salvador
SY Syria
TH Thailand
TN Tunisia
TR Türkiye
TT Trinidad and Tobago
TW Taiwan
TZ Tanzania
UA Ukraine
UG Uganda
US United States
UY Uruguay
UZ Uzbekistan
VE Venezuela
VN Vietnam
ZA South Africa
ZM Zambia
ZW Zimbabwe`
