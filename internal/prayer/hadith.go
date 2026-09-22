package prayer

import (
	"strings"
	"time"
)

// Offline hadith collection about prayer (salah): instant, zero RAM, zero
// processes — no AI round-trip. Entries prefer a prayer key (fajr/isha/…)
// when tagged, otherwise any entry fits. Rotation is daily so the alert
// shows a fresh one each day.

// hadithEntry is one authentic hadith; prayers tags it to specific prayers
// (empty = fits any prayer).
type hadithEntry struct {
	text    string
	source  string
	prayers []string
}

var hadithsAr = []hadithEntry{
	{
		text:    "بَشِّرِ الْمَشَّائِينَ فِي الظُّلَمِ إِلَى الْمَسَاجِدِ بِالنُّورِ التَّامِّ يَوْمَ الْقِيَامَةِ",
		source:  "رواه أبو داود والترمذي",
		prayers: []string{Fajr, Isha},
	},
	{
		text:    "إِنَّ أَثْقَلَ صَلَاةٍ عَلَى الْمُنَافِقِينَ صَلَاةُ الْعِشَاءِ وَصَلَاةُ الْفَجْرِ",
		source:  "متفق عليه",
		prayers: []string{Fajr, Isha},
	},
	{
		text:    "مَنْ صَلَّى الْبَرْدَيْنِ دَخَلَ الْجَنَّةَ",
		source:  "متفق عليه",
		prayers: []string{Fajr, Asr},
	},
	{
		text:   "أَرَأَيْتُمْ لَوْ أَنَّ نَهْرًا بِبَابِ أَحَدِكُمْ يَغْتَسِلُ مِنْهُ كُلَّ يَوْمٍ خَمْسَ مَرَّاتٍ، هَلْ يَبْقَى مِنْ دَرَنِهِ شَيْءٌ؟ قَالُوا: لَا يَبْقَى مِنْ دَرَنِهِ شَيْءٌ، قَالَ: فَذَلِكَ مَثَلُ الصَّلَوَاتِ الْخَمْسِ، يَمْحُو اللَّهُ بِهِنَّ الْخَطَايَا",
		source: "متفق عليه",
	},
	{
		text:   "الْعَهْدُ الَّذِي بَيْنَنَا وَبَيْنَهُمُ الصَّلَاةُ، فَمَنْ تَرَكَهَا فَقَدْ كَفَرَ",
		source: "رواه أحمد والترمذي والنسائي",
	},
	{
		text:   "إِنَّ أَوَّلَ مَا يُحَاسَبُ بِهِ الْعَبْدُ يَوْمَ الْقِيَامَةِ مِنْ عَمَلِهِ صَلَاتُهُ، فَإِنْ صَلُحَتْ فَقَدْ أَفْلَحَ وَأَنْجَحَ، وَإِنْ فَسَدَتْ فَقَدْ خَابَ وَخَسِرَ",
		source: "رواه الترمذي",
	},
	{
		text:   "الصَّلَوَاتُ الْخَمْسُ، وَالْجُمُعَةُ إِلَى الْجُمُعَةِ، كَفَّارَةٌ لِمَا بَيْنَهُنَّ مَا لَمْ تُغْشَ الْكَبَائِرُ",
		source: "رواه مسلم",
	},
	{
		text:   "صَلَاةُ الْجَمَاعَةِ تَفْضُلُ صَلَاةَ الْفَذِّ بِسَبْعٍ وَعِشْرِينَ دَرَجَةً",
		source: "متفق عليه",
	},
	{
		text:   "الْمَلَائِكَةُ تُصَلِّي عَلَى أَحَدِكُمْ مَا دَامَ فِي مُصَلَّاهُ: اللَّهُمَّ اغْفِرْ لَهُ، اللَّهُمَّ ارْحَمْهُ",
		source: "متفق عليه",
	},
	{
		text:   "مَنْ حَافَظَ عَلَيْهَا كَانَتْ لَهُ نُورًا وَبُرْهَانًا وَنَجَاةً يَوْمَ الْقِيَامَةِ",
		source: "رواه أحمد",
	},
}

var hadithsEn = []hadithEntry{
	{
		text:    "Give glad tidings to those who walk to the mosques in darkness: complete light on the Day of Resurrection.",
		source:  "Abu Dawud & Tirmidhi",
		prayers: []string{Fajr, Isha},
	},
	{
		text:    "The heaviest prayers for the hypocrites are Isha and Fajr.",
		source:  "Bukhari & Muslim",
		prayers: []string{Fajr, Isha},
	},
	{
		text:    "Whoever prays the two cool prayers (Fajr and Asr) will enter Paradise.",
		source:  "Bukhari & Muslim",
		prayers: []string{Fajr, Asr},
	},
	{
		text:   "If there were a river at one's door in which he bathed five times a day, would any dirt remain? That is the likeness of the five prayers: Allah wipes away sins through them.",
		source: "Bukhari & Muslim",
	},
	{
		text:   "The covenant between us and them is prayer; whoever abandons it has disbelieved.",
		source: "Ahmad, Tirmidhi & Nasa'i",
	},
	{
		text:   "The first thing a person will be questioned about on the Day of Resurrection is prayer; if it is sound, he succeeds, and if corrupt, he fails.",
		source: "Tirmidhi",
	},
	{
		text:   "The five prayers and Friday to Friday expiate what is between them, so long as major sins are avoided.",
		source: "Muslim",
	},
	{
		text:   "Congregational prayer exceeds praying alone by twenty-seven degrees.",
		source: "Bukhari & Muslim",
	},
	{
		text:   "The angels keep praying for one of you while he remains in his place of prayer: O Allah, forgive him; O Allah, have mercy on him.",
		source: "Bukhari & Muslim",
	},
	{
		text:   "Whoever guards the prayer, it will be light, proof, and salvation for him on the Day of Resurrection.",
		source: "Ahmad",
	},
}

// HadithFor returns a hadith about prayer in lang ("ar" or "en"),
// preferring ones tagged for prayerKey and rotating daily.
func HadithFor(prayerKey, lang string, at time.Time) (text, source string) {
	list := hadithsEn
	if strings.ToLower(strings.TrimSpace(lang)) == "ar" {
		list = hadithsAr
	}
	key := strings.ToLower(strings.TrimSpace(prayerKey))
	var tagged []hadithEntry
	for _, h := range list {
		for _, p := range h.prayers {
			if p == key {
				tagged = append(tagged, h)
				break
			}
		}
	}
	pool := list
	if len(tagged) > 0 {
		pool = tagged
	}
	if len(pool) == 0 {
		return "", ""
	}
	h := pool[at.YearDay()%len(pool)]
	return h.text, h.source
}
