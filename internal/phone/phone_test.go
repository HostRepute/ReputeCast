package phone

import "testing"

func TestParse(t *testing.T) {
	cases := []struct {
		raw, region     string
		e164, country   string
		display, masked string
		typ             LineType
		wantErr         error
	}{
		{"+1 312 555 0123", "", "+13125550123", "US", "+1 312 555 0123", "+1 312 ••• 0123", Unknown, nil},
		{"(312) 555-0187", "US", "+13125550187", "US", "+1 312 555 0187", "+1 312 ••• 0187", Unknown, nil},
		{"+1 888 555 0142", "", "+18885550142", "US", "+1 888 555 0142", "+1 888 ••• 0142", TollFree, nil},
		{"+44 7700 900456", "", "+447700900456", "GB", "+44 7700 900456", "+44 7700 ••• 0456", Unknown, nil},
		{"07700 900456", "GB", "+447700900456", "GB", "+44 7700 900456", "+44 7700 ••• 0456", Unknown, nil},
		{"0044 7700 900456", "", "+447700900456", "GB", "+44 7700 900456", "+44 7700 ••• 0456", Unknown, nil},
		{"+49 151 23452207", "", "+4915123452207", "DE", "+49 1512 3452207", "+49 1512 ••• 2207", Mobile, nil},
		{"+1 416 555 0199", "", "+14165550199", "CA", "+1 416 555 0199", "+1 416 ••• 0199", Unknown, nil},
		{"+353 85 123 4567", "", "+353851234567", "IE", "+353 85 123 4567", "+353 85 ••• 4567", Mobile, nil},
		{"555 0123", "", "", "", "", "", "", ErrNoCountry},
		{"hello", "US", "", "", "", "", "", ErrInvalid},
		{"+1 2", "", "", "", "", "", "", ErrInvalid},
	}
	for _, c := range cases {
		n, err := Parse(c.raw, c.region)
		if c.wantErr != nil {
			if err != c.wantErr {
				t.Errorf("Parse(%q) error %v, want %v", c.raw, err, c.wantErr)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): %v", c.raw, err)
			continue
		}
		if n.E164 != c.e164 || n.Country != c.country || n.Display != c.display || n.Type != c.typ {
			t.Errorf("Parse(%q) = %+v, want %s %s %q %s", c.raw, n, c.e164, c.country, c.display, c.typ)
		}
		if m := Mask(n.E164); m != c.masked {
			t.Errorf("Mask(%s) = %q, want %q", n.E164, m, c.masked)
		}
	}
}

func TestTimeZone(t *testing.T) {
	cases := []struct {
		e164   string
		zone   string
		source ZoneSource
	}{
		// Chicago's 312 maps to one zone.
		{"+13125550123", "America/Chicago", ZoneAreaCode},
		// San Francisco's 415.
		{"+14155550177", "America/Los_Angeles", ZoneAreaCode},
		// A toll-free number is in no zone: the country's main zone.
		{"+18885550142", "America/New_York", ZoneCountry},
		{"+447700900456", "Europe/London", ZoneCountry},
		{"+4915123452207", "Europe/Berlin", ZoneAreaCode},
	}
	for _, c := range cases {
		z, src := TimeZone(c.e164)
		if z != c.zone || src != c.source {
			t.Errorf("TimeZone(%s) = %s, %s; want %s, %s", c.e164, z, src, c.zone, c.source)
		}
	}
	if z, src := RecipientZone("America/Denver", "+13125550123"); z != "America/Denver" || src != ZoneContact {
		t.Errorf("the contact's zone wins: got %s %s", z, src)
	}
	if z, src := RecipientZone("Mars/Olympus", "+13125550123"); z != "America/Chicago" || src != ZoneAreaCode {
		t.Errorf("an invalid contact zone falls back to the number: got %s %s", z, src)
	}
}
