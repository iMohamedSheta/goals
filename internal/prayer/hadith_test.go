package prayer

import (
	"strings"
	"testing"
	"time"
)

func TestHadithFor(t *testing.T) {
	day := time.Date(2026, 9, 22, 12, 0, 0, 0, time.UTC)
	ar, srcAr := HadithFor(Fajr, "ar", day)
	if strings.TrimSpace(ar) == "" || strings.TrimSpace(srcAr) == "" {
		t.Fatalf("empty arabic hadith: %q / %q", ar, srcAr)
	}
	en, srcEn := HadithFor("dhuhr", "en", day)
	if strings.TrimSpace(en) == "" || strings.TrimSpace(srcEn) == "" {
		t.Fatalf("empty english hadith: %q / %q", en, srcEn)
	}
	// Deterministic per day: same inputs give the same hadith.
	ar2, _ := HadithFor(Fajr, "ar", day)
	if ar2 != ar {
		t.Fatalf("not deterministic: %q vs %q", ar, ar2)
	}
	// Language isolation: arabic text must not equal english text.
	if ar == en {
		t.Fatalf("ar/en collision: %q", ar)
	}
	// Unknown prayer still yields a general hadith, never empty.
	g, _ := HadithFor("sunrise", "en", day)
	if strings.TrimSpace(g) == "" {
		t.Fatalf("empty fallback hadith")
	}
}
