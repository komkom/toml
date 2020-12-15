package toml

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"testing"
	"unicode"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseXX(t *testing.T) {

	tests := []struct {
		doc string
		err string
	}{
		{
			doc: `[[x.y.z]]
					key=1
					[x]
					y=2`,
			err: `attempt to redefine a key`,
		},
		{
			doc: `[table]
				[table]`,
			err: `table attempt to redefine a key`,
		},
		{
			doc: `
						uyt=1
						[[a]]
						x=1
						[[a]]
						x=1
						[a.b]
						u=2
						[[a]]
						[[a.b]]
						u=1
						[[a.b]]
						u=1
						[i.y]
						o.o.o=1`,
		},
		{
			doc: `
					[[a]]
					x=1
					[[a]]
					x=1`,
		},
	}

	for idx, ts := range tests {

		t.Log(`idx`, idx, `doc`, ts.doc)

		buf := bytes.NewBufferString(ts.doc + "\n")
		f := NewFilter()

		_, err := io.Copy(f, buf)

		f.WriteRune(EOF)

		if ts.err != `` {
			require.Error(t, err)
			require.Contains(t, err.Error(), ts.err)
			continue
		}

		require.NoError(t, err)

		f.WriteRune('\n')
		require.Equal(t, 0, len(f.state.scopes))
		f.Close()

		t.Log(`json`, string(f.state.buf.Bytes()))

		ok := json.Valid(f.state.buf.Bytes())
		require.True(t, ok)
	}

}

func TestParse(t *testing.T) {

	tests := []struct {
		doc string
		err string
	}{
		{
			doc: `key=""`,
		},
		{
			doc: `key=''`,
		},
		{
			doc: `
			x=1
			y=2
			z=3`,
		},
		{
			doc: `
				uyt=1
				[[a]]
				x=1
				[[a]]
				x=1
				[a.b]
				u=2
				[[a]]
				[[a.b]]
				u=1
				[[a.b]]
				u=1
				[i.y]
				o.o.o=1`,
		},
		{
			doc: `[t]
			x=1
			[z]
			k.j=1`,
		},

		{
			doc: `[t1]
			x=1
			y=2
			[t2]
			x=2
			[t3]
			x=3
			r=4`,
		},
		{
			doc: `
				[[arr]]
				x=1
				y=2
				[[arr]]
				x=3
				y=2
				[x]
									`,
		},
		{
			doc: `
			y=1
			u=2
			o.i.o='test'
			p=[1,2,{x=1}]`,
		},

		{
			doc: `
			#[tt]
			x.u.i=1
			u=3
			y=2`,
		},

		{
			doc: `
			key={x.u.i=1,u=3,y=2}
									`,
		},

		{
			doc: `[[arr.b]]
			x=1
			y=2`,
		},

		{
			doc: `[[arr]]
			key=1
			bb=true
			[[arr]]
			#v1=1

			[[arr.b.c]]

			[table]
			x=1
			y=2
			z="""test xx"""
			uu='''test'''`,
		},
		{
			doc: `[[arr.b]]
			key=1
			[[arr.b]]
			[x]
			[[arr.b.c]]`,
			err: `array attempt to redefine a key`,
		},

		{
			doc: `[[arr.b]]
			key=1
			[[arr.b]]
			[arx]
			x=2`,
		},
		{
			doc: `[[x.y.z]]
				key=1
				[x]
				y=2`,
			err: `attempt to redefine a key`,
		},
		{
			doc: `[table]
				[table]`,
			err: `table attempt to redefine a key`,
		},
		{
			doc: `[[array]]
			[[array]]`,
		},
		{
			doc: `
			x.y.z = 1
			[x]
			[x.y]
			z=2`,
			err: `attempt to redefine a key`,
		},

		{
			doc: `
			[x.y]
			z=1
			[x]
			y.z = 1`,
			err: `attempt to redefine a key`,
		},
		{
			doc: `
			[x.y]
			z=1
			[[x.y.z]]`,
			err: `attempt to redefine a key`,
		},
		{
			doc: `
			[x.y]
			z=1
			[[x.y]]
			z=2`,
			err: `attempt to redefine a key`,
		},
		{
			doc: `key = { x.y.z = 1, x.y.z = 1 }`,
			err: `attempt to redefine a key`,
		},
		{
			doc: `key={x=1,y=[
			1, #comment
			2,
			3,
			],o={i=2,o=3}}`,
		},
		{
			doc: `[a.r]
			[a.e]
			`,
		},
		{
			doc: `[a.r]
			[a.r.b]
			[a.r.b.c]
			[a.r.z]
			`,
		},
		{
			doc: `[a.r]
			[a.r.b.i.o]
			[a.r.c]
			[a.e]
			`,
		},
		{
			doc: `[[a.1]]
				[[a.1.2]]
				[a.1.2.3]
				t=1
				[a.1.2.3.4]
				t=1
				[a.1.2.5]
				t=1

				[[a.1.2]]
				[[a.1.2.3]]
				[[a.1]]
				`,
		},
		{
			doc: `date= 1975-12-12T00:00:00-07:10`,
		},
		{
			doc: `date= 1975-12-12T00:00:00+07:10`,
		},
		{
			doc: `date= 1975-12-12 00:00:00-07:10`,
		},
		{
			doc: `date= {x=1975-12-12,y=1}`,
		},
		{
			doc: `date= 1975-12-12
							key=1`,
		},
		{
			doc: `date= 1975-12-12T00:00:00Z`,
		},
		{
			doc: `date= 1975-12-12`,
		},
		{
			doc: `date= 1975-12-122`,
			err: `invalid character after value`,
		},
		{
			doc: `date= 0975-12-12`,
		},
		{
			doc: `date= 12:10:00`,
		},
		{
			doc: `date= 09:10:00`,
		},
		{
			doc: `date= 09:15:00.99999`,
		},
		{
			doc: `date= 1975-12-40`,
			err: `invalid number of days in month`,
		},
		{
			doc: `date= 1975-14-11`,
			err: `invalid month in date`,
		},
		{
			doc: `date= 1976-02-30`,
			err: `invalid number of days in month`,
		},
		{
			doc: `date= 1976-02-180`,
			err: `invalid character after value`,
		},

		{
			doc: `key = " \uxxxx  " `,
			err: `invalid digit`,
		},
		{
			doc: `key=true`,
		},
		{
			doc: `key=false`,
		},
		{
			doc: `key=truee`,
			err: `invalid character after value`,
		},
		{
			doc: `key=tru`,
			err: `invalid boolean`,
		},
		{
			doc: `key=[
								true,false,
								true,false
								,
											]`,
		},
		{
			doc: `key="xx#commentxx"
											key2=1`,
		},
		{
			doc: `key="xxxx"#comment
												key2=1`,
		},
		{
			doc: `key = [1,2,
											3, #comment
											#comment
											4,]`,
		},
		{
			doc: `[table]#comment`,
		},
		{
			doc: `[[table]]#comment`,
		},
		{
			doc: `key="xxxx"#comment`,
		},
		{
			doc: `key=123#comment`,
		},

		{
			doc: `key="xxxx"`,
		},
		{
			doc: `key = "xx \b\t xx"
																				    	`,
		},
		{
			doc: `	key = "value"`,
		},
		{
			doc: `key = "value"
																																key2 = "value2"`,
		},
		{
			doc: `key = "xx \ xx"
																															`,
			err: `quoted string has unescaped backslash`,
		},

		{
			doc: `key = """value"""`,
		},
		{
			doc: `key = """value  ""\\"
																															"""`,
		},
		{
			doc: `key = """value  \
																																"""`,
		},
		{
			doc: `key = 'value'`,
		},
		{
			doc: `key = 'value
																															'`,
			err: `\n found in literal string`,
		},
		{
			doc: `key = '/some/path'`,
		},
		{
			doc: `key = '''value''' `,
		},
		{
			doc: `key = ''''value''''`,
		},
		{
			doc: `key = ''''value 1
																													2'
																													3'
																													4
																													5''''`,
		},
		{
			doc: `key = ''''value'''''`,
		},
		{
			doc: `key = """"value""""`,
		},
		{
			doc: `key = ''''value''''
																														key2 = 'test'`,
		},
		{
			doc: `ke y = 'value'`,
			err: `invalid space in key`,
		},
		{
			doc: `key 2 = "value"`,
			err: `invalid space in key`,
		},
		{
			doc: `key2 = "value"
			k ey = 'value'`,
			err: `invalid space in key`,
		},
		{
			doc: `key = """ \uD9FF  """ `,
		},
		{
			doc: `key = """ \uxxxx  """ `,
			err: `invalid digit`,
		},
		{
			doc: `key = " \\\uD9FF  " `,
		},
		{
			doc: `key = " \uxxxx  " `,
			err: `invalid digit`,
		},
		{
			doc: `key = 123`,
		},
		{
			doc: `key = +123`,
		},
		{
			doc: `key = -123`,
		},
		{
			doc: `key = +0
																											key2 = -0`,
		},

		{
			doc: `key = 0xDEFF`,
		},
		{
			doc: `key = 0o1234567`,
		},
		{
			doc: `key = 0b100101011100`,
		},
		{
			doc: `key = +0.123e10`,
		},
		{
			doc: `key = 100_000.0`,
		},
		{
			doc: `test."k".i.ey = 100_000.0`,
		},
		{
			doc: `"test".k.i.ey = 100_000.0`,
		},
		{
			doc: `"test".k.'ff i'.ey = 100_000.0`,
		},

		{
			doc: `test..k = 1`,
			err: `invalid '.' in key`,
		},
		{
			doc: `"test"k = 1`,
			err: `invalid character in key after quote`,
		},
		{
			doc: `xx"yy".zz = 1`,
			err: `invalid character before '"'`,
		},
		{
			doc: `xx."yy"'.zz' = 1`,
			err: `invalid character in key after quote`,
		},
		{
			doc: `"" = 1`,
		},
		{
			doc: `[ table ]`,
		},
		{
			doc: `[ table ]
																		key = 1
																				`,
		},
		{
			doc: `[ "t".able ]
																				`,
		},
		{
			doc: `key = { v.alue = {test2=1}, value2 = "2" }
																		`,
		},
		{
			doc: `key = {value=1}`,
		},

		{
			doc: `key = [1,2,3]`,
		},
		{
			doc: `key = [1,[2,3],4]`,
		},
		{
			doc: `key = []`,
		},
		{
			doc: `key = {}`,
		},

		{
			doc: `[[ array ]]`,
		},
		{
			doc: `[[ array ]]
					[ table ]`,
		},

		{
			doc: `[[array]]`,
		},
		{
			doc: `[[array]]
					key2=2
					[table]
					key1=1`,
		},
		{
			doc: `[table]
					[xable2]`,
		},

		{
			doc: `[table]
					key=1
					key1=2`,
		},
		{
			doc: `[table]]`,
			err: `invalid character at table end`,
		},
		{
			doc: `[[array]`,
			err: `array end invalid`,
		},

		{
			doc: `[[array]]]`,
			err: `array end invalid`,
		},
		{
			doc: `key =[
																				1,
																				2,
																				3,
																							]`,
		},
		{
			doc: `key = {
																				}`,
			err: `inline table contains \n`,
		},

		{
			doc: `key.test2.'test2'."xxx\uAAAAx" = 'value'`,
		},
		{
			doc: `[table."xx\UAAAAAAx".'xx']`,
		},
		{
			doc: `key = {x.y."z..\uAAAA.y.y.y"=1, u=2}
							key2 = 2`,
		},
		{
			doc: `[[array.x.y.z]]`,
		},
		{
			doc: `key = 0`,
		},
		{
			doc: `key = { key1= 't', key2 = [
																		1,
																		2,
																		0o1234,
																					]}`,
		},
		{
			doc: `key = ["1",2,'2',0x123]`,
		},
		{
			doc: `key = {value=0x123}`,
		},
	}

	for idx, ts := range tests {

		t.Log(`idx`, idx, `doc`, ts.doc)

		buf := bytes.NewBufferString(ts.doc + "\n")
		f := NewFilter()

		_, err := io.Copy(f, buf)

		f.WriteRune(EOF)

		if ts.err != `` {
			require.Error(t, err)
			require.Contains(t, err.Error(), ts.err)
			continue
		}

		require.NoError(t, err)

		f.WriteRune('\n')
		require.Equal(t, 0, len(f.state.scopes))
		f.Close()

		t.Log(`json`, string(f.state.buf.Bytes()))

		ok := json.Valid(f.state.buf.Bytes())
		require.True(t, ok)
	}
}

func TestUnicode(t *testing.T) {

	tests := []struct {
		code         string
		expectedCode string
		short        bool
		err          string
	}{
		{
			code:         `D7FFF`,
			expectedCode: `D7FF`,
			short:        true,
		},
		{
			code:  `D7FF`,
			short: true,
		},
		{
			code:  `FFFF`,
			short: true,
		},
		{
			code:         `d7FF`,
			expectedCode: `D7FF`,
			short:        true,
		},
		{
			code:  `g7FF`,
			short: true,
			err:   `invalid digit`,
		},

		{
			code:         `D7FF16X`,
			expectedCode: `D7FF16`,
		},
		{
			code: `D7FF16`,
		},
		{
			code: `D7FF17`,
			err:  `invalid code`,
		},
		{
			code:         `d7ff16`,
			expectedCode: `D7FF16`,
		},
		{
			code: `G7HH17`,
			err:  `invalid digit`,
		},
	}

	for _, ts := range tests {

		t.Run(ts.code, func(t *testing.T) {

			var pf ParseFunc
			pf = Unicode
			if ts.short {
				pf = ShortUnicode
			}
			state := State{buf: &bytes.Buffer{}}
			state.PushScope(pf, StringType, nil)
			sc := state.PeekScope()

			var err error
			for _, r := range ts.code {
				err = sc.Parse(r, &state)
				if err != nil {
					break
				}
				if len(state.scopes) == 0 {
					break
				}
			}

			if ts.err != `` {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ts.err)
				return
			}

			require.NoError(t, err)

			if ts.expectedCode != `` {
				assert.Equal(t, ts.expectedCode, string(state.buf.Bytes()))
			} else {
				assert.Equal(t, ts.code, string(state.buf.Bytes()))
			}
		})
	}
}

func TestPrefixNumber(t *testing.T) {

	tests := []struct {
		number         string
		expectedNumber string
		ranges         []*unicode.RangeTable
		err            string
	}{
		{
			number:         `D7FFD`,
			expectedNumber: `D7FFD`,
			ranges:         hexRanges,
		},
		{
			number:         `D7FFe`,
			expectedNumber: `D7FFE`,
			ranges:         hexRanges,
		},
		{
			number: ` `,
			ranges: hexRanges,
			err:    `empty number`,
		},
		{
			number:         `1467`,
			expectedNumber: `1467`,
			ranges:         octalRanges,
		},
		{
			number: `1468`,
			ranges: octalRanges,
			err:    `invalid character in number`,
		},
		{
			number: ` `,
			ranges: octalRanges,
			err:    `empty number`,
		},
		{
			number:         `101001100`,
			expectedNumber: `101001100`,
			ranges:         binRanges,
		},
		{
			number: `1002`,
			ranges: binRanges,
			err:    `invalid character in number`,
		},
		{
			number: ` `,
			ranges: binRanges,
			err:    `empty number`,
		},
	}

	for _, ts := range tests {

		t.Log(`number`, ts.number)

		var pf ParseFunc
		pf = PrefixNumber(ts.ranges)
		state := State{buf: &bytes.Buffer{}}
		state.PushScope(pf, OtherType, nil)
		sc := state.PeekScope()

		var err error
		for _, r := range ts.number {

			err = sc.Parse(r, &state)
			if err != nil {
				break
			}
			if len(state.scopes) == 0 {
				break
			}
		}

		if ts.err != `` {
			require.Error(t, err)
			assert.Contains(t, err.Error(), ts.err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, ts.expectedNumber, string(state.buf.Bytes()))
	}
}

func TestFloat(t *testing.T) {

	tests := []struct {
		float         string
		expectedFloat string
		err           string
	}{
		{
			float:         `1234`,
			expectedFloat: `1234`,
		},
		{
			float:         `0.1234`,
			expectedFloat: `0.1234`,
		},
		{
			float:         `12e6`,
			expectedFloat: `12e6`,
		},
		{
			float:         `+0.1234`,
			expectedFloat: `0.1234`,
		},
		{
			float:         `-0.1234`,
			expectedFloat: `-0.1234`,
		},
		{
			float:         `-1.1234E-12`,
			expectedFloat: `-1.1234E-12`,
		},
		{
			float:         `1_2_3_4`,
			expectedFloat: `1234`,
		},
		{
			float:         `0.1_23_4`,
			expectedFloat: `0.1234`,
		},
		{
			float:         `12e6_8`,
			expectedFloat: `12e68`,
		},
		{
			float:         `-1.1234E-1_2_3`,
			expectedFloat: `-1.1234E-123`,
		},
		{
			float: `1_2_3_4_`,
			err:   `invalid float ending`,
		},
		{
			float: `1_2__3_4`,
			err:   `invalid '_' in float`,
		},
		{
			float: `1_e100e200`,
			err:   `invalid 'e' in float`,
		},
		{
			float: `1e_100`,
			err:   `invalid '_' in float`,
		},
		{
			float: `1_e100`,
			err:   `invalid 'e' in float`,
		},
		{
			float: `+_100`,
			err:   `invalid '_' in float`,
		},
		{
			float: `01`,
			err:   `invalid float character after zero`,
		},
		{
			float:         `100_000.0`,
			expectedFloat: `100000.0`,
		},
		{
			float:         `0.0`,
			expectedFloat: `0.0`,
		},
		{
			float:         `1`,
			expectedFloat: `1`,
		},
	}

	for _, ts := range tests {

		t.Run(ts.float, func(t *testing.T) {

			state := State{buf: &bytes.Buffer{}}
			state.PushScope(Float(OtherState, nil, 0), OtherType, nil)
			sc := state.PeekScope()

			var err error
			for _, r := range ts.float {

				err = sc.Parse(r, &state)
				if err != nil {
					break
				}
				if len(state.scopes) == 0 {
					break
				}
			}

			if err == nil {
				if len(state.scopes) > 0 {
					etmp := sc.Parse('\n', &state)
					if etmp != nil && !errors.Is(etmp, ErrDontAdvance) {
						err = etmp
					}
				}
			}

			if ts.err != `` {
				require.Error(t, err)
				assert.Contains(t, err.Error(), ts.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, ts.expectedFloat, string(state.buf.Bytes()))
		})
	}
}

func TestParseTypes(t *testing.T) {

	tests := []struct {
		doc      string
		expected string
		err      string
	}{
		{
			doc:      `key=""`,
			expected: `{"key":""}`,
		},
		{
			doc:      `key="a bcd"`,
			expected: `{"key":"a bcd"}`,
		},
		{
			doc: `key = """value  \

uu"""`,
			expected: `{"key":"value  uu"}`,
		},
		{
			doc:      `key = """"""`,
			expected: `{"key":""}`,
		},
		{
			doc:      `key = """"" """`,
			expected: `{"key":"\"\" "}`,
		},
		{
			doc:      `key = """"""""`,
			expected: `{"key":"\"\""}`,
		},
		{
			doc:      `key = """" """"`,
			expected: `{"key":"\" \""}`,
		},
		{
			doc:      `key = """value  \"""""`,
			expected: `{"key":"value  \"\""}`,
		},
		{
			doc:      `key = """value  \""""`,
			expected: `{"key":"value  \""}`,
		},
		{
			doc: `key = """value
second2
""""`,
			expected: `{"key":"value\nsecond2\n\""}`,
		},
		{
			doc:      `key = """\uABCD"""`,
			expected: `{"key":"\uABCD"}`,
		},
		{
			doc: `key = """\uABCD\

xx"""`,
			expected: `{"key":"\uABCDxx"}`,
		},

		{
			doc:      `key = """\UD7FF16"""`,
			expected: `{"key":"\\UD7FF16"}`,
		},
		{
			doc: `key = """\UD7FF16\
xx"""`,
			expected: `{"key":"\\UD7FF16xx"}`,
		},
		{
			doc:      `key = 'test'`,
			expected: `{"key":"test"}`,
		},
		{
			doc:      `key = '"test"'`,
			expected: `{"key":"\"test\""}`,
		},
		{
			doc: `key = 'x	x'`,
			expected: `{"key":"x\tx"}`,
		},
		{
			doc:      `key = 'x"x'`,
			expected: `{"key":"x\"x"}`,
		},
		{
			doc:      `key = ''''''`,
			expected: `{"key":""}`,
		},
		{
			doc:      `key = '''''''`,
			expected: `{"key":"'"}`,
		},
		{
			doc:      `key = ''''x'''`,
			expected: `{"key":"'x"}`,
		},
		{
			doc:      `key = '''x''''`,
			expected: `{"key":"x'"}`,
		},
		{
			doc:      `key = 1978-01-01`,
			expected: `{"key":"1978-01-01"}`,
		},
		{
			doc:      `key = 10:00:00`,
			expected: `{"key":"10:00:00"}`,
		},
		{
			doc:      `key = 10:00:00`,
			expected: `{"key":"10:00:00"}`,
		},
		{
			doc:      `key= 1965-05-27T07:32:00Z`,
			expected: `{"key":"1965-05-27T07:32:00Z"}`,
		},
		{
			doc:      `key=1979-05-27T00:32:00-07:00`,
			expected: `{"key":"1979-05-27T00:32:00-07:00"}`,
		},
		{
			doc:      `key=1979-05-27T00:32:00.999999-07:00`,
			expected: `{"key":"1979-05-27T00:32:00.999999-07:00"}`,
		},
		{
			doc:      `key=1979-05-27 07:32:00Z`,
			expected: `{"key":"1979-05-27 07:32:00Z"}`,
		},
		{
			doc:      `key=1979-05-27T07:32:00`,
			expected: `{"key":"1979-05-27T07:32:00"}`,
		},
		{
			doc:      `key=1979-05-27T00:32:00.999999`,
			expected: `{"key":"1979-05-27T00:32:00.999999"}`,
		},
		{
			doc:      `key=1979-05-27`,
			expected: `{"key":"1979-05-27"}`,
		},
		{
			doc:      `key= 07:32:00`,
			expected: `{"key":"07:32:00"}`,
		},
		{
			doc:      `key=00:32:00.999999`,
			expected: `{"key":"00:32:00.999999"}`,
		},
		{
			doc:      `flag=true`,
			expected: `{"flag":true}`,
		},
		{
			doc:      `flag=false`,
			expected: `{"flag":false}`,
		},
		{
			doc:      `key = 12345`,
			expected: `{"key":12345}`,
		},
		{
			doc:      `key = 12345.9089`,
			expected: `{"key":12345.9089}`,
		},
		{
			doc:      `key = 12345.9089e10`,
			expected: `{"key":12345.9089e10}`,
		},
		{
			doc:      `key = -12345.9089e10`,
			expected: `{"key":-12345.9089e10}`,
		},
		{
			doc:      `key = +12345.9089e10`,
			expected: `{"key":12345.9089e10}`,
		},
		{
			doc:      `key = +12345.9089e+10`,
			expected: `{"key":12345.9089e+10}`,
		},
		{
			doc:      `key = +12345.9089e-10`,
			expected: `{"key":12345.9089e-10}`,
		},
		{
			doc:      `key = 0.0`,
			expected: `{"key":0.0}`,
		},
		{
			doc:      `key = 0.1`,
			expected: `{"key":0.1}`,
		},
		{
			doc:      `key = 0xD7FFD`,
			expected: `{"key":"0xD7FFD"}`,
		},
		{
			doc:      `key = 0o12345`,
			expected: `{"key":"0o12345"}`,
		},
		{
			doc:      `key = 0o1100101`,
			expected: `{"key":"0o1100101"}`,
		},
	}

	for idx, ts := range tests {

		t.Log(`idx`, idx, `doc`, ts.doc)

		buf := bytes.NewBufferString(ts.doc + "\n")
		f := NewFilter()

		_, err := io.Copy(f, buf)

		f.WriteRune(EOF)

		if ts.err != `` {
			require.Error(t, err)
			assert.Contains(t, err.Error(), ts.err)
			continue
		}

		require.NoError(t, err)

		f.WriteRune('\n')
		assert.Equal(t, 0, len(f.state.scopes))
		f.Close()

		ok := json.Valid(f.state.buf.Bytes())
		assert.True(t, ok)
		assert.Equal(t, ts.expected, string(f.state.buf.Bytes()))
	}
}
