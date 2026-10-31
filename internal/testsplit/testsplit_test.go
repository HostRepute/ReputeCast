package testsplit

import "testing"

func TestApplySetsTheLineForThePackage(t *testing.T) {
	env := "m/other run ^(TestX)$\nm/pkg skip ^(TestA|TestB)$\n"
	got := map[string]string{}
	set := func(name, value string) error { got[name] = value; return nil }
	if err := apply(env, "m/pkg", set); err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got["test.skip"] != "^(TestA|TestB)$" {
		t.Fatalf("set %v, want only test.skip for m/pkg", got)
	}
}

func TestApplyLeavesOtherPackagesAlone(t *testing.T) {
	for _, env := range []string{"", "m/other run ^(TestX)$"} {
		if err := apply(env, "m/pkg", func(name, _ string) error {
			t.Fatalf("set %s with %q", name, env)
			return nil
		}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestApplyRefusesABadLine(t *testing.T) {
	for _, env := range []string{"m/pkg ^(TestX)$", "m/pkg only ^(TestX)$", "m/pkg run a b"} {
		if err := apply(env, "m/pkg", func(string, string) error { return nil }); err == nil {
			t.Fatalf("accepted %q", env)
		}
	}
}
