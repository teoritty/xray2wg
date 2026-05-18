package transport

import "testing"

func TestRegistry_resolvesCanonicalAndAlias(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"tcp", "tcp"},
		{"TCP", "tcp"},
		{"ws", "ws"},
		{"WebSocket", "ws"},
		{"grpc", "grpc"},
		{"GUN", "grpc"},
	}
	for _, c := range cases {
		got, err := Default.Resolve(c.input)
		if err != nil {
			t.Fatalf("%q: %v", c.input, err)
		}
		if got.Name() != c.want {
			t.Fatalf("%q: got %q, want %q", c.input, got.Name(), c.want)
		}
	}
}

func TestRegistry_unknownTransportReturnsError(t *testing.T) {
	if _, err := Default.Resolve("xhttp"); err == nil {
		t.Fatal("expected unknown-transport error for xhttp at this point in the migration")
	}
	if _, err := Default.Resolve(""); err == nil {
		t.Fatal("expected error for empty transport name")
	}
}
