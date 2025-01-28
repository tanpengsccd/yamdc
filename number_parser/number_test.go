package number_parser

import (
	"os"
	"testing"
	"yamdc/model"

	"github.com/stretchr/testify/assert"
)

func TestNumber(t *testing.T) {
	os.Args = []string{"cmd", "-config", "../config.json"}
	checkList := map[string]*Number{
		"huPE@18P2P.XV544.avi": {
			NumberId: "XV-544",
		},
		// "iptd-899-B": {
		// 	NumberId: "IPTD-899",
		// 	Episode:  "B",
		// },
		// "641_0124_01.MP4": {
		// 	NumberId: "641-0124", //流出的番号,几乎查不到
		// 	Episode:  "01",
		// },
		// "Max Girls 15  Haruka Ito Marimi Natsusaki Arisa Kuroki Aino Kishi Cecil Fujisaki Rio [XV723].avi": {
		// 	NumberId: "XV-723",
		// },
		// "052624_01-CD3-C.mp4": {
		// 	NumberId: "052624-01",
		// 	Episode:  "3",
		// 	IsCnSub:  true,
		// },
		// "052624_01_cd3_c.mp4": {
		// 	NumberId: "052624-01",
		// 	Episode:  "3",
		// 	IsCnSub:  true,
		// },
		// "k0009-c_cd1-4k.mp4": {
		// 	NumberId: "K0009",
		// 	Episode:  "1",
		// 	IsCnSub:  true,
		// },
		// "n001-Cd1-4k.mp4": {
		// 	NumberId:     "N001",
		// 	IsUncensored: true,
		// 	Episode:      "1",
		// 	Is4k:         true,
		// },

		// // "abc11-leak-c.mp4": {
		// // 	NumberId: "ABC-11",
		// // 	IsCnSub:  true,
		// // 	Episode:  "C",
		// // },
		// "HEYZO-3332.mp4": {
		// 	NumberId:     "HEYZO-3332",
		// 	IsUncensored: true,
		// },
		// "052624_01.mp4": (&Number{}).WithNumberId("052624-01"),
		// "【更多福利www.51fuliku.com-福利库】IDBD278-01.mp4": model.DefaultNumber().WithNumberId("IDBD-278").WithEpisode("01"),
		// "100118-all.wmv": model.DefaultNumber(),
		// "javset.com-srxv353.wmv": {
		// 	NumberId: "SRXV-353",
		// },
		// // "052624_01-C.mp4": {
		// // 	NumberId: "052624-01",
		// // 	IsCnSub:  true,
		// // },
		// "052624_01-CD2.mp4": {
		// 	NumberId: "052624-01",

		// 	Episode: "2",
		// },
	}

	for file, expectInfo := range checkList {
		rs, err := ParseWithFileName(file)
		assert.NoError(t, err)
		assert.Equal(t, expectInfo.NumberId, rs.NumberId, "NumberId file:%s", file)
		assert.Equal(t, expectInfo.IsCnSub, rs.IsCnSub, "IsCnSub file:%s", file)
		assert.Equal(t, expectInfo.Episode, rs.Episode, "Episode file:%s", file)
		// assert.Equal(t, expectInfo., rs.GetIsUncensorMovie(), "info:%s", file)
		// assert.Equal(t, expectInfo.GetIs4k(), rs.GetIs4k())
	}
}

func TestCategory(t *testing.T) {
	{
		n, err := Parse("fc2-ppv-12345")
		assert.NoError(t, err)
		assert.Equal(t, 1, len(n.GetCategory()))
		assert.Equal(t, model.CatFC2, n.GetCategory()[0])
	}
	{
		n, err := Parse("abc-0001")
		assert.NoError(t, err)
		assert.Equal(t, 0, len(n.GetCategory()))
	}
}

func TestAlnumber(t *testing.T) {
	assert.Equal(t, "fc2ppv12345", GetCleanID("fc2-ppv_12345"))
}
