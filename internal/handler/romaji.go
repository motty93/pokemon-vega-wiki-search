package handler

import (
	"strings"
	"unicode/utf8"
)

// romajiToKatakana はローマ字入力の場合のみカタカナに変換する。
// 変換できない場合は空文字を返す。
func romajiToKatakana(s string) string {
	if s == "" {
		return ""
	}
	lower := strings.ToLower(s)

	// ASCII英字のみで構成されているか確認
	for _, r := range lower {
		if r < 'a' || r > 'z' {
			return ""
		}
	}

	var result strings.Builder
	i := 0
	for i < len(lower) {
		// 促音: 同じ子音が連続 (kk, tt, pp, ss, etc.)
		if i+1 < len(lower) && lower[i] == lower[i+1] && !isVowel(lower[i]) && lower[i] != 'n' {
			result.WriteString("ッ")
			i++
			continue
		}

		// 最長一致: 4文字 → 3文字 → 2文字 → 1文字
		matched := false
		for l := 4; l >= 1; l-- {
			if i+l > len(lower) {
				continue
			}
			chunk := lower[i : i+l]
			if kana, ok := romajiTable[chunk]; ok {
				result.WriteString(kana)
				i += l
				matched = true
				break
			}
		}
		if !matched {
			// 変換できない文字があれば、末尾の未変換部分は無視して
			// 変換済み部分だけ返す（前方一致検索なので途中まででOK）
			break
		}
	}

	if result.Len() == 0 {
		return ""
	}
	return result.String()
}

func isVowel(c byte) bool {
	return c == 'a' || c == 'i' || c == 'u' || c == 'e' || c == 'o'
}

// hiraganaToKatakana はひらがなをカタカナに変換する。
// ひらがなが含まれていなければ空文字を返す。
func hiraganaToKatakana(s string) string {
	if s == "" {
		return ""
	}
	hasHiragana := false
	var result strings.Builder
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		// ひらがな範囲: U+3041〜U+3096
		if r >= 0x3041 && r <= 0x3096 {
			result.WriteRune(r + 0x60) // カタカナ = ひらがな + 0x60
			hasHiragana = true
		} else {
			result.WriteRune(r)
		}
		i += size
	}
	if !hasHiragana {
		return ""
	}
	return result.String()
}

var romajiTable = map[string]string{
	// 母音
	"a": "ア", "i": "イ", "u": "ウ", "e": "エ", "o": "オ",
	// か行
	"ka": "カ", "ki": "キ", "ku": "ク", "ke": "ケ", "ko": "コ",
	// さ行
	"sa": "サ", "si": "シ", "su": "ス", "se": "セ", "so": "ソ",
	"shi": "シ",
	// た行
	"ta": "タ", "ti": "チ", "tu": "ツ", "te": "テ", "to": "ト",
	"chi": "チ", "tsu": "ツ",
	// な行
	"na": "ナ", "ni": "ニ", "nu": "ヌ", "ne": "ネ", "no": "ノ",
	// は行
	"ha": "ハ", "hi": "ヒ", "hu": "フ", "he": "ヘ", "ho": "ホ",
	"fu": "フ",
	// ま行
	"ma": "マ", "mi": "ミ", "mu": "ム", "me": "メ", "mo": "モ",
	// や行
	"ya": "ヤ", "yu": "ユ", "yo": "ヨ",
	// ら行
	"ra": "ラ", "ri": "リ", "ru": "ル", "re": "レ", "ro": "ロ",
	// わ行
	"wa": "ワ", "wi": "ウィ", "we": "ウェ", "wo": "ヲ",
	// ん
	"n'": "ン", "nn": "ン",
	// が行
	"ga": "ガ", "gi": "ギ", "gu": "グ", "ge": "ゲ", "go": "ゴ",
	// ざ行
	"za": "ザ", "zi": "ジ", "zu": "ズ", "ze": "ゼ", "zo": "ゾ",
	"ji": "ジ",
	// だ行
	"da": "ダ", "di": "ヂ", "du": "ヅ", "de": "デ", "do": "ド",
	// ば行
	"ba": "バ", "bi": "ビ", "bu": "ブ", "be": "ベ", "bo": "ボ",
	// ぱ行
	"pa": "パ", "pi": "ピ", "pu": "プ", "pe": "ペ", "po": "ポ",
	// 拗音 (きゃ等)
	"kya": "キャ", "kyi": "キィ", "kyu": "キュ", "kye": "キェ", "kyo": "キョ",
	"sha": "シャ", "shu": "シュ", "she": "シェ", "sho": "ショ",
	"sya": "シャ", "syu": "シュ", "syo": "ショ",
	"cha": "チャ", "chu": "チュ", "che": "チェ", "cho": "チョ",
	"tya": "チャ", "tyu": "チュ", "tyo": "チョ",
	"nya": "ニャ", "nyi": "ニィ", "nyu": "ニュ", "nye": "ニェ", "nyo": "ニョ",
	"hya": "ヒャ", "hyi": "ヒィ", "hyu": "ヒュ", "hye": "ヒェ", "hyo": "ヒョ",
	"mya": "ミャ", "myi": "ミィ", "myu": "ミュ", "mye": "ミェ", "myo": "ミョ",
	"rya": "リャ", "ryi": "リィ", "ryu": "リュ", "rye": "リェ", "ryo": "リョ",
	"gya": "ギャ", "gyi": "ギィ", "gyu": "ギュ", "gye": "ギェ", "gyo": "ギョ",
	"ja": "ジャ", "ju": "ジュ", "je": "ジェ", "jo": "ジョ",
	"jya": "ジャ", "jyu": "ジュ", "jyo": "ジョ",
	"zya": "ジャ", "zyu": "ジュ", "zyo": "ジョ",
	"bya": "ビャ", "byi": "ビィ", "byu": "ビュ", "bye": "ビェ", "byo": "ビョ",
	"pya": "ピャ", "pyi": "ピィ", "pyu": "ピュ", "pye": "ピェ", "pyo": "ピョ",
	"dya": "ヂャ", "dyi": "ヂィ", "dyu": "ヂュ", "dye": "ヂェ", "dyo": "ヂョ",
	// 特殊
	"fa": "ファ", "fi": "フィ", "fe": "フェ", "fo": "フォ",
	"va": "ヴァ", "vi": "ヴィ", "vu": "ヴ", "ve": "ヴェ", "vo": "ヴォ",
	"tsa": "ツァ", "tsi": "ツィ", "tse": "ツェ", "tso": "ツォ",
	// 外来音（ティ、ディ等）
	"thi": "ティ", "tha": "テャ", "thu": "テュ", "the": "テェ", "tho": "テョ",
	"dhi": "ディ", "dha": "デャ", "dhu": "デュ", "dhe": "デェ", "dho": "デョ",
	// ー（長音）
	"-": "ー",
}
