package localbus

import "testing"

func TestHostFromEnvDefault(t *testing.T) {
	t.Setenv("BUS_HOST", "")
	if got := HostFromEnv(); got != DefaultHost {
		t.Fatalf("HostFromEnv()=%q, want %q", got, DefaultHost)
	}
}

func TestAddrUsesBusHost(t *testing.T) {
	t.Setenv("BUS_HOST", "127.0.0.2")
	if got, want := Addr(8080), "127.0.0.2:8080"; got != want {
		t.Fatalf("Addr()=%q, want %q", got, want)
	}
}

func TestHTTPURLUsesBusHost(t *testing.T) {
	t.Setenv("BUS_HOST", "127.0.0.2")
	if got, want := HTTPURL(8081, "local/v1"), "http://127.0.0.2:8081/local/v1"; got != want {
		t.Fatalf("HTTPURL()=%q, want %q", got, want)
	}
}
