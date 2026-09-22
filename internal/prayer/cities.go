package prayer

// City is a selectable prayer location (coordinates are city-center
// approximations — good to about a minute for prayer times).
type City struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	NameAr  string  `json:"nameAr"`
	Country string  `json:"country"`
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	TZ      string  `json:"tz"`
}

// Cities lists the built-in selectable locations, Egypt first (user's home),
// then the wider world.
var Cities = []City{
	// ---------- Egypt (Africa/Cairo) ----------
	{ID: "cairo", Name: "Cairo", NameAr: "القاهرة", Country: "Egypt", Lat: 30.0444, Lng: 31.2357, TZ: "Africa/Cairo"},
	{ID: "giza", Name: "Giza", NameAr: "الجيزة", Country: "Egypt", Lat: 30.0131, Lng: 31.2089, TZ: "Africa/Cairo"},
	{ID: "october", Name: "6th of October", NameAr: "6 أكتوبر", Country: "Egypt", Lat: 29.9627, Lng: 30.9429, TZ: "Africa/Cairo"},
	{ID: "alex", Name: "Alexandria", NameAr: "الإسكندرية", Country: "Egypt", Lat: 31.2001, Lng: 29.9187, TZ: "Africa/Cairo"},
	{ID: "matruh", Name: "Marsa Matruh", NameAr: "مرسى مطروح", Country: "Egypt", Lat: 31.3543, Lng: 27.2373, TZ: "Africa/Cairo"},
	{ID: "siwa", Name: "Siwa", NameAr: "سيوة", Country: "Egypt", Lat: 29.2032, Lng: 25.5195, TZ: "Africa/Cairo"},
	{ID: "portsaid", Name: "Port Said", NameAr: "بورسعيد", Country: "Egypt", Lat: 31.2653, Lng: 32.3019, TZ: "Africa/Cairo"},
	{ID: "suez", Name: "Suez", NameAr: "السويس", Country: "Egypt", Lat: 29.9668, Lng: 32.5498, TZ: "Africa/Cairo"},
	{ID: "ismailia", Name: "Ismailia", NameAr: "الإسماعيلية", Country: "Egypt", Lat: 30.5965, Lng: 32.2715, TZ: "Africa/Cairo"},
	{ID: "damietta", Name: "Damietta", NameAr: "دمياط", Country: "Egypt", Lat: 31.4165, Lng: 31.8133, TZ: "Africa/Cairo"},
	{ID: "mansoura", Name: "Mansoura", NameAr: "المنصورة", Country: "Egypt", Lat: 31.0379, Lng: 31.3815, TZ: "Africa/Cairo"},
	{ID: "tanta", Name: "Tanta", NameAr: "طنطا", Country: "Egypt", Lat: 30.7865, Lng: 31.0004, TZ: "Africa/Cairo"},
	{ID: "zagazig", Name: "Zagazig", NameAr: "الزقازيق", Country: "Egypt", Lat: 30.5877, Lng: 31.5020, TZ: "Africa/Cairo"},
	{ID: "shebin", Name: "Shibin El Kom", NameAr: "شبين الكوم", Country: "Egypt", Lat: 30.5590, Lng: 31.0092, TZ: "Africa/Cairo"},
	{ID: "banha", Name: "Banha", NameAr: "بنها", Country: "Egypt", Lat: 30.4681, Lng: 31.1848, TZ: "Africa/Cairo"},
	{ID: "damanhur", Name: "Damanhur", NameAr: "دمنهور", Country: "Egypt", Lat: 30.8303, Lng: 31.0100, TZ: "Africa/Cairo"},
	{ID: "kafr", Name: "Kafr El Sheikh", NameAr: "كفر الشيخ", Country: "Egypt", Lat: 31.1107, Lng: 30.9388, TZ: "Africa/Cairo"},
	{ID: "hurghada", Name: "Hurghada", NameAr: "الغردقة", Country: "Egypt", Lat: 27.2579, Lng: 33.8116, TZ: "Africa/Cairo"},
	{ID: "sharm", Name: "Sharm El Sheikh", NameAr: "شرم الشيخ", Country: "Egypt", Lat: 27.9158, Lng: 34.3300, TZ: "Africa/Cairo"},
	{ID: "fayoum", Name: "Fayoum", NameAr: "الفيوم", Country: "Egypt", Lat: 29.3084, Lng: 30.8428, TZ: "Africa/Cairo"},
	{ID: "benisuef", Name: "Beni Suef", NameAr: "بني سويف", Country: "Egypt", Lat: 29.0661, Lng: 31.0994, TZ: "Africa/Cairo"},
	{ID: "minya", Name: "Minya", NameAr: "المنيا", Country: "Egypt", Lat: 28.1099, Lng: 30.7503, TZ: "Africa/Cairo"},
	{ID: "asyut", Name: "Asyut", NameAr: "أسيوط", Country: "Egypt", Lat: 27.1783, Lng: 31.1859, TZ: "Africa/Cairo"},
	{ID: "sohag", Name: "Sohag", NameAr: "سوهاج", Country: "Egypt", Lat: 26.5570, Lng: 31.6948, TZ: "Africa/Cairo"},
	{ID: "qena", Name: "Qena", NameAr: "قنا", Country: "Egypt", Lat: 26.1642, Lng: 32.7267, TZ: "Africa/Cairo"},
	{ID: "luxor", Name: "Luxor", NameAr: "الأقصر", Country: "Egypt", Lat: 25.6872, Lng: 32.6396, TZ: "Africa/Cairo"},
	{ID: "aswan", Name: "Aswan", NameAr: "أسوان", Country: "Egypt", Lat: 24.0889, Lng: 32.8998, TZ: "Africa/Cairo"},
	{ID: "kharga", Name: "Kharga (New Valley)", NameAr: "الخارجة", Country: "Egypt", Lat: 25.4516, Lng: 30.5460, TZ: "Africa/Cairo"},
	{ID: "arish", Name: "Arish", NameAr: "العريش", Country: "Egypt", Lat: 31.1319, Lng: 33.7984, TZ: "Africa/Cairo"},
	// ---------- Middle East ----------
	{ID: "makkah", Name: "Mecca", NameAr: "مكة المكرمة", Country: "Saudi Arabia", Lat: 21.4225, Lng: 39.8262, TZ: "Asia/Riyadh"},
	{ID: "madinah", Name: "Medina", NameAr: "المدينة المنورة", Country: "Saudi Arabia", Lat: 24.5247, Lng: 39.5692, TZ: "Asia/Riyadh"},
	{ID: "riyadh", Name: "Riyadh", NameAr: "الرياض", Country: "Saudi Arabia", Lat: 24.7136, Lng: 46.6753, TZ: "Asia/Riyadh"},
	{ID: "jeddah", Name: "Jeddah", NameAr: "جدة", Country: "Saudi Arabia", Lat: 21.4858, Lng: 39.1925, TZ: "Asia/Riyadh"},
	{ID: "dubai", Name: "Dubai", NameAr: "دبي", Country: "UAE", Lat: 25.2048, Lng: 55.2708, TZ: "Asia/Dubai"},
	{ID: "abudhabi", Name: "Abu Dhabi", NameAr: "أبوظبي", Country: "UAE", Lat: 24.4539, Lng: 54.3773, TZ: "Asia/Dubai"},
	{ID: "doha", Name: "Doha", NameAr: "الدوحة", Country: "Qatar", Lat: 25.2854, Lng: 51.5310, TZ: "Asia/Qatar"},
	{ID: "kuwait", Name: "Kuwait City", NameAr: "مدينة الكويت", Country: "Kuwait", Lat: 29.3759, Lng: 47.9774, TZ: "Asia/Kuwait"},
	{ID: "manama", Name: "Manama", NameAr: "المنامة", Country: "Bahrain", Lat: 26.2285, Lng: 50.5860, TZ: "Asia/Bahrain"},
	{ID: "muscat", Name: "Muscat", NameAr: "مسقط", Country: "Oman", Lat: 23.5880, Lng: 58.3829, TZ: "Asia/Muscat"},
	{ID: "amman", Name: "Amman", NameAr: "عمّان", Country: "Jordan", Lat: 31.9539, Lng: 35.9106, TZ: "Asia/Amman"},
	{ID: "jerusalem", Name: "Jerusalem", NameAr: "القدس", Country: "Palestine", Lat: 31.7683, Lng: 35.2137, TZ: "Asia/Jerusalem"},
	{ID: "beirut", Name: "Beirut", NameAr: "بيروت", Country: "Lebanon", Lat: 33.8938, Lng: 35.5018, TZ: "Asia/Beirut"},
	{ID: "baghdad", Name: "Baghdad", NameAr: "بغداد", Country: "Iraq", Lat: 33.3152, Lng: 44.3661, TZ: "Asia/Baghdad"},
	{ID: "istanbul", Name: "Istanbul", NameAr: "إسطنبول", Country: "Türkiye", Lat: 41.0082, Lng: 28.9784, TZ: "Europe/Istanbul"},
	{ID: "khartoum", Name: "Khartoum", NameAr: "الخرطوم", Country: "Sudan", Lat: 15.5007, Lng: 32.5599, TZ: "Africa/Khartoum"},
	{ID: "tripoli", Name: "Tripoli", NameAr: "طرابلس", Country: "Libya", Lat: 32.8872, Lng: 13.1913, TZ: "Africa/Tripoli"},
	{ID: "tunis", Name: "Tunis", NameAr: "تونس", Country: "Tunisia", Lat: 36.7992, Lng: 10.1815, TZ: "Africa/Tunis"},
	{ID: "algiers", Name: "Algiers", NameAr: "الجزائر", Country: "Algeria", Lat: 36.7538, Lng: 3.0588, TZ: "Africa/Algiers"},
	{ID: "casablanca", Name: "Casablanca", NameAr: "الدار البيضاء", Country: "Morocco", Lat: 33.5731, Lng: -7.5898, TZ: "Africa/Casablanca"},
	// ---------- World ----------
	{ID: "london", Name: "London", NameAr: "لندن", Country: "UK", Lat: 51.5074, Lng: -0.1278, TZ: "Europe/London"},
	{ID: "paris", Name: "Paris", NameAr: "باريس", Country: "France", Lat: 48.8566, Lng: 2.3522, TZ: "Europe/Paris"},
	{ID: "berlin", Name: "Berlin", NameAr: "برلين", Country: "Germany", Lat: 52.5200, Lng: 13.4050, TZ: "Europe/Berlin"},
	{ID: "nyc", Name: "New York", NameAr: "نيويورك", Country: "USA", Lat: 40.7128, Lng: -74.0060, TZ: "America/New_York"},
	{ID: "toronto", Name: "Toronto", NameAr: "تورونتو", Country: "Canada", Lat: 43.6532, Lng: -79.3832, TZ: "America/Toronto"},
	{ID: "la", Name: "Los Angeles", NameAr: "لوس أنجلوس", Country: "USA", Lat: 34.0522, Lng: -118.2437, TZ: "America/Los_Angeles"},
	{ID: "saopaulo", Name: "São Paulo", NameAr: "ساو باولو", Country: "Brazil", Lat: -23.5558, Lng: -46.6396, TZ: "America/Sao_Paulo"},
	{ID: "jakarta", Name: "Jakarta", NameAr: "جاكرتا", Country: "Indonesia", Lat: -6.2088, Lng: 106.8456, TZ: "Asia/Jakarta"},
	{ID: "kl", Name: "Kuala Lumpur", NameAr: "كوالالمبور", Country: "Malaysia", Lat: 3.1390, Lng: 101.6869, TZ: "Asia/Kuala_Lumpur"},
	{ID: "delhi", Name: "Delhi", NameAr: "دلهي", Country: "India", Lat: 28.6139, Lng: 77.2090, TZ: "Asia/Kolkata"},
	{ID: "karachi-city", Name: "Karachi", NameAr: "كراتشي", Country: "Pakistan", Lat: 24.8607, Lng: 67.0011, TZ: "Asia/Karachi"},
	{ID: "dhaka", Name: "Dhaka", NameAr: "دكا", Country: "Bangladesh", Lat: 23.8103, Lng: 90.4125, TZ: "Asia/Dhaka"},
}

// FindCity returns the city with the given id, or nil.
func FindCity(id string) *City {
	for i := range Cities {
		if Cities[i].ID == id {
			return &Cities[i]
		}
	}
	return nil
}
