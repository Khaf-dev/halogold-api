package service

// PriceProvider menyediakan harga emas terkini (rupiah per gram).
// Dibuat sebagai interface agar mudah di-mock saat unit test, dan agar
// sumber harga bisa diganti (config statis -> API eksternal) tanpa mengubah
// logic bisnis di GoldService.
type PriceProvider interface {
	CurrentPrice() int64
}

// StaticPriceProvider mengembalikan harga tetap dari konfigurasi.
// Sesuai BRD, sumber harga untuk test bersifat dummy.
type StaticPriceProvider struct {
	price int64
}

func NewStaticPriceProvider(price int64) *StaticPriceProvider {
	return &StaticPriceProvider{price: price}
}

func (p *StaticPriceProvider) CurrentPrice() int64 {
	return p.price
}
