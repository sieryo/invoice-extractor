package buyer

type Buyer struct {
	Name string `json:"name"`

	// NPWP format lama (15 digit, tanpa leading 0)
	NPWP15 string `json:"npwp_15,omitempty"`

	// NPWP format baru (16 digit, biasanya 0 + NPWP15)
	NPWP16 string `json:"npwp_16,omitempty"`

	// NITKU (Nomor Identitas Tempat Kegiatan Usaha)
	NITKU string `json:"nitku,omitempty"`

	Email   string `json:"email,omitempty"`
	Address string `json:"address"`
}

func (b *Buyer) PrimaryTaxID() string {
	if b.NPWP16 != "" {
		return b.NPWP16
	}
	if b.NPWP15 != "" {
		return b.NPWP15
	}
	return b.NITKU
}

func (b *Buyer) TKU() string {
	return b.NITKU
}
