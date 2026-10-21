package queuelog

import (
	"errors"
	"testing"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Entry
	}{
		{
			name: "connect",
			line: "1729500000|1729499990.42|support|SIP/101|CONNECT|10|1729499990.43|3\n",
			want: Entry{Time: 1729500000, UniqueID: "1729499990.42", Queue: "support", Agent: "SIP/101", Event: "CONNECT", Data1: "10", Data2: "1729499990.43", Data3: "3"},
		},
		{
			name: "missing data fields",
			line: "1729500000|NONE|NONE|NONE|QUEUESTART",
			want: Entry{Time: 1729500000, UniqueID: "NONE", Queue: "NONE", Agent: "NONE", Event: "QUEUESTART"},
		},
		{
			name: "extra fields stay in data3",
			line: "1729500000|1.1|q|SIP/1|COMPLETECALLER|5|60|1|extra\r\n",
			want: Entry{Time: 1729500000, UniqueID: "1.1", Queue: "q", Agent: "SIP/1", Event: "COMPLETECALLER", Data1: "5", Data2: "60", Data3: "1|extra"},
		},
		{
			name: "special characters are kept verbatim",
			line: "1729500000|1.2|sales 100%|Agent/O'Brien|ENTERQUEUE||\"Bob\" <123>|1",
			want: Entry{Time: 1729500000, UniqueID: "1.2", Queue: "sales 100%", Agent: "Agent/O'Brien", Event: "ENTERQUEUE", Data2: "\"Bob\" <123>", Data3: "1"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.line)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got != tt.want {
				t.Errorf("got  %+v\nwant %+v", got, tt.want)
			}
		})
	}
}

func TestParseSkips(t *testing.T) {
	for _, line := range []string{"", "\n", "garbage", "2024-10-21 10:00:00|x|q|a|CONNECT", "-1|x|q|a|CONNECT", "99999999999999999999|x|q|a|E"} {
		if _, err := Parse(line); !errors.Is(err, ErrSkip) {
			t.Errorf("Parse(%q) err = %v, want ErrSkip", line, err)
		}
	}
}

func TestConvertLocal(t *testing.T) {
	tests := map[string]string{
		"Local/101@from-queue/n": "Agent/101",
		"local/202@ctx":          "Agent/202",
		"Local/303":              "Agent/303",
		"SIP/101":                "SIP/101",
		"NONE":                   "NONE",
		"":                       "",
	}
	for in, want := range tests {
		if got := ConvertLocal(in); got != want {
			t.Errorf("ConvertLocal(%q) = %q, want %q", in, got, want)
		}
	}
}
