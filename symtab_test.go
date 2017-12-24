package args

import (
	"testing"
)

func TestGetNonExistent(t *testing.T) {
	table := newTestingSymtab('$')
	v, err := table.get("a")
	if err != nil {
		t.Errorf(`unexpected error: %v`, err)
	}
	if v != nil {
		t.Errorf(`found "%s", expected nothing`, v.s)
	}
}

func TestPutFirstWins(t *testing.T) {
	table := newTestingSymtab('$')
	table.put("$a0", "1")
	table.put("$a0", "2")
	expected := "1"
	v, _ := table.get("a0")
	if v.s != expected {
		t.Errorf(`found "%s", expected "%s"`, v.s, expected)
	}
	// do it again and use debugger to follow execution path
	// (watch: no need to resolve the second time)
	v, _ = table.get("a0")
	if v.s != "1" {
		t.Errorf(`found "%s", expected "%s"`, v.s, expected)
	}
}

func TestGetUnresolvedValue(t *testing.T) {
	table := newTestingSymtab('$')
	table.put("$a1", "$[b1]")
	expected := "$[b1]"
	v, _ := table.get("a1")
	if v.s != expected {
		t.Errorf(`found "%s", expected "%s"`, v.s, expected)
	}
}

func TestGetResolvedValue(t *testing.T) {
	table := newTestingSymtab('$')
	table.put("$a2", "a $[b2] e")
	table.put("$b2", "b $[c2] d")
	table.put("$c2", "C")
	expected := "a b C d e"
	v, _ := table.get("a2")
	if v.s != expected {
		t.Errorf(`found "%s", expected "%s"`, v.s, expected)
	}
}

func TestGetCycle(t *testing.T) {
	table := newTestingSymtab('$')
	table.put("$a3", "a $[b3] e")
	table.put("$b3", "b $[c3] d")
	table.put("$c3", "$[a3]")
	expected := `cyclical symbol definition detected: "a3"`
	// defer panicHandler(expected, t)
	v, err := table.get("a3")
	if err == nil {
		t.Errorf(`expected error missing, expected: "%s" value: "%s"`, expected, v.s)
	} else if err.Error() != expected {
		t.Errorf(`unexpected error: "%s", expected: "%s"`, err.Error(), expected)
	}

}

func TestSymbols1(t *testing.T) {
	table1 := newTestingSymtab('$')
	test := func(name1, name2 string, expectOkay bool) {
		table1.put(name1, "...")
		v, _ := table1.get(name2)
		if expectOkay {
			if v == nil || v.s != "..." {
				t.Errorf(`failed to get "%s", which was put as "%s"`, name2, name1)
			}
		} else {
			if v != nil {
				t.Errorf(`could unexpectedly get "%s", which was put as "%s"`, name2, name1)
			}
		}
	}
	test("$foo", "foo", true)
	test("$$bar", "bar", false)
	test("$$", "", false)
	test("$", "", false)
	test("aaa", "", false)
	test("", "", false)
}

func TestSymbols2(t *testing.T) {
	table1 := newTestingSymtab('⌘')
	test := func(name1, name2 string, expectOkay bool) {
		table1.put(name1, "...")
		v, _ := table1.get(name2)
		if expectOkay {
			if v == nil || v.s != "..." {
				t.Errorf(`failed to get "%s", which was put as "%s"`, name2, name1)
			}
		} else {
			if v != nil {
				t.Errorf(`could unexpectedly get "%s", which was put as "%s"`, name2, name1)
			}
		}
	}
	test("⌘foo", "foo", true)
	test("⌘⌘bar", "bar", false)
	test("⌘⌘", "", false)
	test("⌘", "", false)
	test("aaa", "", false)
	test("", "", false)
}

func newTestingSymtab(prefix rune) symtab {
	c := NewConfig()
	c.SetSpecial(SpecSymbolPrefix, prefix)
	return newSymtab(c)
}
