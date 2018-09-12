package url

import (
	"testing"
)

type parseTestCase struct {
	raw      string
	fail     bool
	expected *URL
}

func (c *parseTestCase) run(t *testing.T) {
	// Attempt to parse.
	url, err := Parse(c.raw)
	if err != nil {
		if !c.fail {
			t.Fatal("parsing failed when it should have succeeded:", err)
		}
		return
	} else if c.fail {
		t.Fatal("parsing should have failed but did not")
	}

	// Verify protocol.
	if url.Protocol != c.expected.Protocol {
		t.Error("protocol mismatch:", url.Protocol, "!=", c.expected.Protocol)
	}

	// Verify username.
	if url.Username != c.expected.Username {
		t.Error("username mismatch:", url.Username, "!=", c.expected.Username)
	}

	// Verify hostname.
	if url.Hostname != c.expected.Hostname {
		t.Error("hostname mismatch:", url.Hostname, "!=", c.expected.Hostname)
	}

	// Verify port.
	if url.Port != c.expected.Port {
		t.Error("port mismatch:", url.Port, "!=", c.expected.Port)
	}

	// Verify path.
	if url.Path != c.expected.Path {
		t.Error("path mismatch:", url.Path, "!=", c.expected.Path)
	}
}

func TestParseEmptyInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "",
		fail: true,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "",
		},
	}
	test.run(t)
}

func TestParseEmptyHostnameAndPathInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  ":",
		fail: true,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "",
		},
	}
	test.run(t)
}

func TestParseEmptyHostnameInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  ":path",
		fail: true,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "",
		},
	}
	test.run(t)
}

func TestParseUsernameEmptyHostnameInvalid(t *testing.T) {
	test := parseTestCase{
		raw:  "user@:path",
		fail: true,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "",
		},
	}
	test.run(t)
}

func TestParsePath(t *testing.T) {
	test := parseTestCase{
		raw:  "/this/is/a:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "/this/is/a:path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnameIsLocal(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host",
		fail: false,
		expected: &URL{
			Protocol: Protocol_Local,
			Username: "",
			Hostname: "",
			Port:     0,
			Path:     "user@host",
		},
	}
	test.run(t)
}

func TestParseHostnameEmptyPath(t *testing.T) {
	test := parseTestCase{
		raw:  "host:",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "",
			Hostname: "host",
			Port:     0,
			Path:     "",
		},
	}
	test.run(t)
}

func TestParseHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw:  "host:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "",
			Hostname: "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnamePathWithColonAtStart(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host::path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     ":path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnamePathWithColonInMiddle(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:pa:th",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "pa:th",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnamePathWithColonAtEnd(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:path:",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "path:",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnameWithAtPath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@ho@st:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "ho@st",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnamePathWithAt(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:pa@th",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "pa@th",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnamePortPath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:65535:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     65535,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnameZeroPortPath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:0:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnameDoubleZeroPortPath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:00:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnameNumericPath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:65536:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "65536:path",
		},
	}
	test.run(t)
}

func TestParseUsernameHostnameHexNumericPath(t *testing.T) {
	test := parseTestCase{
		raw:  "user@host:aaa:path",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "user",
			Hostname: "host",
			Port:     0,
			Path:     "aaa:path",
		},
	}
	test.run(t)
}

func TestParseUnicodeUsernameHostnamePath(t *testing.T) {
	test := parseTestCase{
		raw:  "üsér@høst:пат",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "üsér",
			Hostname: "høst",
			Port:     0,
			Path:     "пат",
		},
	}
	test.run(t)
}

func TestParseUnicodeUsernameHostnamePortPath(t *testing.T) {
	test := parseTestCase{
		raw:  "üsér@høst:23:пат",
		fail: false,
		expected: &URL{
			Protocol: Protocol_SSH,
			Username: "üsér",
			Hostname: "høst",
			Port:     23,
			Path:     "пат",
		},
	}
	test.run(t)
}

func TestParseCustomInvalidScheme(t *testing.T) {
	test := parseTestCase{
		raw:  "5http://example.org/",
		fail: true,
	}
	test.run(t)
}

func TestParseCustom(t *testing.T) {
	test := parseTestCase{
		raw: "htt-...+p://example.org/",
		expected: &URL{
			Protocol: Protocol_Custom,
			Path:     "htt-...+p://example.org/",
		},
	}
	test.run(t)
}
