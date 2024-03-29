package toml

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode"

	"github.com/pkg/errors"
)

var (
	ErrDontAdvance    = fmt.Errorf(`dont-advance`)
	ErrNotImplemented = fmt.Errorf(`not-implemented`)
	EOF               = '\255' // invalid utf 8 character

	bin = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     48,
				Hi:     49,
				Stride: 1,
			},
		},
	}

	binRanges = []*unicode.RangeTable{
		bin,
	}

	octal = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     48,
				Hi:     55,
				Stride: 1,
			},
		},
	}

	octalRanges = []*unicode.RangeTable{
		octal,
	}

	digits = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     48,
				Hi:     57,
				Stride: 1,
			},
		},
	}

	letters = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     65,
				Hi:     90,
				Stride: 1,
			},
			{
				Lo:     97,
				Hi:     122,
				Stride: 1,
			},
		},
	}

	dash = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     45,
				Hi:     45,
				Stride: 1,
			},
		},
	}

	underscore = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     95,
				Hi:     95,
				Stride: 1,
			},
		},
	}

	bareRanges = []*unicode.RangeTable{
		digits,
		letters,
		dash,
		underscore,
	}

	hex = &unicode.RangeTable{
		R16: []unicode.Range16{
			{
				Lo:     65,
				Hi:     70,
				Stride: 1,
			},
			{
				Lo:     97,
				Hi:     102,
				Stride: 1,
			},
		},
	}

	hexRanges = []*unicode.RangeTable{
		digits,
		hex,
	}

	digitRanges = []*unicode.RangeTable{
		{
			R16: []unicode.Range16{
				{
					Lo:     48,
					Hi:     57,
					Stride: 1,
				},
			},
		},
	}

	notAllowedStringRanges = []*unicode.RangeTable{
		{
			R16: []unicode.Range16{
				{ // backspace
					Lo:     8,
					Hi:     8,
					Stride: 1,
				},

				{ // \n
					Lo:     10,
					Hi:     10,
					Stride: 1,
				},

				{ // form feed
					Lo:     12,
					Hi:     12,
					Stride: 1,
				},
			},
		},
	}

	allowedMultiStringRanges = []*unicode.RangeTable{
		{
			R16: []unicode.Range16{
				{ // tab & \n
					Lo:     9,
					Hi:     10,
					Stride: 1,
				},
				{ // form feed
					Lo:     12,
					Hi:     12,
					Stride: 1,
				},
				{ // space
					Lo:     32,
					Hi:     32,
					Stride: 1,
				},
			},
		},
	}

	infRunes   = []rune(`inf`)
	nanRunes   = []rune(`nan`)
	trueRunes  = []rune(`true`)
	falseRunes = []rune(`false`)

	escapeCharacters = []rune{'\\', 'b', 't', 'n', 'f', 'r', '"'}

	daysInMonth = []int{31 /*jan*/, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
)

func toJSONString(r rune) string {

	if r == '"' {
		return `\"`
	}

	if r == '\\' {
		return `\\`
	}

	if r == '/' {
		return `\/`
	}

	if r == '\b' {
		return `\b`
	}

	if r == '\f' {
		return `\f`
	}

	if r == '\n' {
		return `\n`
	}

	if r == '\r' {
		return `\r`
	}

	if r == '\t' {
		return `\t`
	}

	return string(r)
}

type Token string

var (
	OTHERT Token
	QQQQT  Token = `""""`
	QQQT   Token = `"""`
	QQT    Token = `""`
	QT     Token = `"`
	SQQQQT Token = `''''`
	SQQQT  Token = `'''`
	SQQT   Token = `''`
	SQT    Token = `'`
	ESCT   Token = `escape`

	OBT      Token = `[`  // open bracket
	OBBT     Token = `[[` // open bracket
	CBT      Token = `]`
	CBBT     Token = `]]`
	CBTSPACE Token = `] `

	DIGITT Token = `digit`
	SIGNT  Token = `sign`
	DOTT   Token = `dot`
	COMT   Token = `comma`

	UNDERT Token = `underscore`
	EXPT   Token = `exp`
	SPACET Token = `space`
)

type ScopeState string

var (
	OtherState ScopeState
	InitState  ScopeState = `init`
	DoneState  ScopeState = `done`

	AfterInitialZeroState ScopeState = `after-initial-zero`
	AfterDotState         ScopeState = `after-dot`
	AfterExpState         ScopeState = `after-exp`
	AfterEOLState         ScopeState = `after-eol`
	AfterEOLReturnState   ScopeState = `after-eol-return`

	AfterTableState ScopeState = `after-table-state`
	AfterArrayState ScopeState = `after-array-state`

	AfterKeyState        ScopeState = `after-key`
	AfterValueState      ScopeState = `after-value`
	AfterFirstValueState ScopeState = `after-first-value`

	AfterQuoteState ScopeState = `after-quote`
	AfterTState     ScopeState = `after-T`
)

type ScopeType string

var (
	OtherType  ScopeType = `other`
	StringType ScopeType = `string`
	KeyType    ScopeType = `key`
)

type NumberType int

const (
	BinNumberType NumberType = iota
	OctalNumberType
	HexNumberType
)

type Filter struct {
	State State
	Buf   *bytes.Buffer
}

func NewFilter() *Filter {

	state := State{
		Buf:  bytes.NewBufferString(`{`),
		defs: MakeDefs(),
	}

	state.PushScope(Top, OtherType, nil)

	return &Filter{Buf: &bytes.Buffer{}, State: state}
}

func (f *Filter) Write(p []byte) (int, error) {

	f.Buf.Write(p)

	for {
		r, _, err := f.Buf.ReadRune()
		if errors.Is(err, io.EOF) {
			break
		}

		if r == '\r' {
			continue
		}

		if r == '\n' {
			f.State.line++
			f.State.position = 0
		} else {
			f.State.position++
		}

		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return 0, parseError(&f.State, `invalid character`)
		}

		if len(f.State.Scopes) == 0 {
			break
		}

		err = f.WriteRune(r)
		if err != nil {
			return 0, err
		}
	}
	f.Buf.Truncate(f.Buf.Len())
	return len(p), nil
}

func (f *Filter) WriteRune(r rune) error {

	for {
		if f.State.inComment {
			if r == '\n' {
				f.State.inComment = false
			}

			if f.State.inComment {
				return nil
			}
		}

		if scidx, ok := f.State.topScopeIdx(); ok && f.State.Scopes[scidx].scopeType == OtherType && r == '#' {
			f.State.inComment = true
			return nil
		}

		scidx, ok := f.State.topScopeIdx()
		if ok {
			err := f.State.Scopes[scidx].Parse(r, &f.State)
			if errors.Is(err, ErrDontAdvance) {
				continue
			}
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func (f *Filter) Close() {
	f.State.defs.keyFilter.Close(f.State.Buf)
	f.State.Buf.WriteRune('}')
}

type Scope struct {
	state     ScopeState
	key       []string
	scopeType ScopeType
	counter   int64
	lastToken Token
	parseFunc ParseFunc
}

func (s *Scope) Parse(r rune, state *State) error {
	return s.parseFunc(r, state, s)
}

type ParseFunc func(r rune, state *State, scope *Scope) error

type State struct {
	Buf       *bytes.Buffer
	Scopes    []Scope
	defs      Defs
	line      int
	position  int
	inComment bool
	keyData   []rune
	data      []rune
}

func (s *State) PushScope(parse ParseFunc, scopeType ScopeType, thisScope *Scope) {

	if thisScope != nil {
		s.Scopes[len(s.Scopes)-1] = *thisScope
	}
	s.Scopes = append(s.Scopes, Scope{parseFunc: parse, scopeType: scopeType})
}

func (s *State) PopScope() {
	if len(s.Scopes) > 0 {
		if s.data != nil {
			s.data = s.data[0:0]
		}
		s.Scopes = s.Scopes[:len(s.Scopes)-1]
	}
}

func (s *State) topScopeIdx() (int, bool) {

	if len(s.Scopes) > 0 {
		return len(s.Scopes) - 1, true
	}
	return 0, false
}

func (s *State) ResetData() {
	if s.keyData != nil {
		s.keyData = s.keyData[0:0]
	}
}

func (s *State) ExtractKeys() []string {
	rawKey := string(s.keyData)
	s.ResetData()
	return strings.Split(rawKey, "\n")
}

func fullKey(key []string) string {
	return strings.Join(key, "\n")
}

func BaseKeyDefs(baseKey []string, insertTable []string, defs Defs) DefineFunc {

	return func(key []string, v Var) bool {
		key = append(baseKey, key...)
		return defs.Define(key, insertTable, v)
	}
}

func validUnicode(code int64) bool {

	if code >= 0 &&
		code <= 0xD7FF16 {
		return true
	}
	if code >= 0xE00016 &&
		code <= 0x10FFFF {
		return true
	}
	return false
}

func ShortUnicode(r rune, state *State, scope *Scope) error {

	if !unicode.IsOneOf(hexRanges, r) {
		return parseError(state, `invalid digit`)
	}

	scope.counter++

	state.data = append(state.data, r)

	r = unicode.ToUpper(r)
	if scope.scopeType == KeyType {
		state.keyData = append(state.keyData, r)
	} else {
		state.Buf.WriteRune(r)
	}

	if scope.counter == 4 {

		v, err := strconv.ParseInt(string(state.data), 16, 64)
		if err != nil {
			return parseError(state, `invalid number`)
		}

		if ok := validUnicode(v); !ok {
			return parseError(state, `invalid code`)
		}

		state.PopScope()
		return nil
	}
	return nil
}

func Unicode(r rune, state *State, scope *Scope) error {

	if !unicode.IsOneOf(hexRanges, r) {
		return parseError(state, `invalid digit`)
	}

	scope.counter++

	state.data = append(state.data, r)

	r = unicode.ToUpper(r)
	if scope.scopeType == KeyType {
		state.keyData = append(state.keyData, r)
	} else {
		state.Buf.WriteRune(r)
	}

	if scope.counter == 6 {

		v, err := strconv.ParseInt(string(state.data), 16, 64)
		if err != nil {
			return parseError(state, `invalid number`)
		}

		if ok := validUnicode(v); !ok {
			return parseError(state, `invalid code`)
		}

		state.PopScope()
		return nil
	}
	return nil
}

func QuotedString(r rune, state *State, scope *Scope) error {

	if unicode.IsOneOf(notAllowedStringRanges, r) {
		return parseError(state, `character not allowed in quoted string`)
	}

	if unicode.IsSpace(r) && r != '\t' && r != ' ' {
		return parseError(state, `character not allowed in quoted string`)
	}

	if scope.lastToken != ESCT && r == '\\' {
		scope.lastToken = ESCT
		return nil
	}

	if r == '"' && scope.lastToken != ESCT {
		state.PopScope()

		if scope.scopeType != KeyType {
			state.Buf.WriteRune('"')
		}
		return nil
	}

	if scope.lastToken == ESCT && r == 'U' {
		if scope.scopeType == KeyType {
			state.keyData = append(state.keyData, '\\')
			state.keyData = append(state.keyData, '\\')
			state.keyData = append(state.keyData, 'U')
		} else {
			state.Buf.WriteString("\\\\U")
		}

		scope.lastToken = OTHERT
		state.PushScope(Unicode, scope.scopeType, scope)
		return nil
	}

	if scope.lastToken == ESCT && r == 'u' {
		if scope.scopeType == KeyType {
			state.keyData = append(state.keyData, '\\')
			state.keyData = append(state.keyData, 'u')
		} else {
			state.Buf.WriteString("\\u")
		}

		scope.lastToken = OTHERT
		state.PushScope(ShortUnicode, scope.scopeType, scope)
		return nil
	}

	if scope.lastToken == ESCT && !isOneOf(r, escapeCharacters) {
		return parseError(state, `quoted string has unescaped backslash`)
	}
	if scope.lastToken == ESCT {
		scope.lastToken = OTHERT
		if scope.scopeType == KeyType {
			state.keyData = append(state.keyData, []rune{'\\', r}...)
		} else {
			state.Buf.WriteRune('\\')
			state.Buf.WriteRune(r)
		}
		return nil
	}
	scope.lastToken = OTHERT
	js := toJSONString(r)
	if scope.scopeType == KeyType {
		state.keyData = append(state.keyData, []rune(js)...)
	} else {
		state.Buf.WriteString(js)
	}
	return nil
}

func TrippleQuotedString(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) && !unicode.IsOneOf(allowedMultiStringRanges, r) {
		return parseError(state, `disallowed character in """`)
	}

	if scope.state == DoneState {
		if r != '"' {
			if scope.lastToken == QQQQT {
				state.Buf.WriteString(`\"`)
			}
			state.Buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.lastToken == QQQT {
			scope.lastToken = QQQQT
			return nil
		}

		state.Buf.WriteString(`\"\""`)
		state.PopScope()
		return nil
	}

	if scope.state == AfterEOLReturnState {
		if !unicode.IsSpace(r) {
			scope.state = InitState
			return ErrDontAdvance
		}

		return nil
	}

	if scope.state == AfterEOLState {

		if r == '\n' {
			scope.state = AfterEOLReturnState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid character in EOL`)
		}
		return nil
	}

	st := scope.state
	if st == OtherState {
		scope.state = InitState
	}

	if st == OtherState && r == '\n' {
		return nil
	}

	if scope.lastToken != ESCT && r == '\\' {

		if scope.lastToken == QQT {
			state.Buf.WriteString(`\"\"`)
		}
		if scope.lastToken == QT {
			state.Buf.WriteString(`\"`)
		}

		scope.lastToken = ESCT
		return nil
	}

	if scope.lastToken == ESCT && r == 'U' {
		scope.lastToken = OTHERT
		state.Buf.WriteString(`\\U`)
		state.PushScope(Unicode, StringType, scope)
		return nil
	}

	if scope.lastToken == ESCT && r == 'u' {
		scope.lastToken = OTHERT
		state.Buf.WriteString(`\u`)
		state.PushScope(ShortUnicode, StringType, scope)
		return nil
	}

	if scope.lastToken == ESCT && unicode.IsSpace(r) {
		scope.lastToken = OTHERT
		scope.state = AfterEOLState
		return ErrDontAdvance
	}

	if scope.lastToken == ESCT && r == '"' {
		scope.lastToken = OTHERT
		state.Buf.WriteString(`\"`)
		return nil
	}

	if scope.lastToken == QQT && r == '"' {
		scope.lastToken = QQQT
		scope.state = DoneState
		return nil
	}

	if scope.lastToken == QT && r == '"' {
		scope.lastToken = QQT
		return nil
	}

	if scope.lastToken != ESCT && r == '"' {
		scope.lastToken = QT
		return nil
	}

	if scope.lastToken == ESCT && !isOneOf(r, escapeCharacters) {
		return parseError(state, `backslash \ not allowed`)
	}

	if scope.lastToken == ESCT {
		state.Buf.WriteRune('\\')
	}

	if scope.lastToken == ESCT && r == '\\' {

		scope.lastToken = OTHERT
		state.Buf.WriteString(`\`)
		return nil
	}

	if scope.lastToken == QT {
		state.Buf.WriteString(`\"`)
	}

	if scope.lastToken == QQT {
		state.Buf.WriteString(`\"\"`)
	}

	scope.lastToken = OTHERT
	state.Buf.WriteString(toJSONString(r))
	return nil
}

func LiteralString(r rune, state *State, scope *Scope) error {

	if r == '\n' {
		return parseError(state, `\n found in literal string`)
	}

	if r == '\'' {
		state.PopScope()

		if scope.scopeType != KeyType {
			state.Buf.WriteRune('"')
		}
		return nil
	}

	js := toJSONString(r)
	if scope.scopeType == KeyType {
		state.keyData = append(state.keyData, []rune(js)...)
	} else {
		state.Buf.WriteString(js)
	}
	return nil
}

func MultiLineLiteralString(r rune, state *State, scope *Scope) error {

	if scope.state == DoneState {
		if r != '\'' {
			if scope.lastToken == SQQQQT {
				state.Buf.WriteRune('\'')
			}

			state.Buf.WriteRune('"')
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.lastToken == SQQQT {
			scope.lastToken = SQQQQT
			return nil
		}

		state.Buf.WriteString(`''"`)
		state.PopScope()
		return nil
	}

	st := scope.state
	if st == OtherState {
		scope.state = InitState
	}

	if st == OtherState && r == '\n' {
		return nil
	}

	if scope.lastToken == SQQT && r == '\'' {
		scope.lastToken = SQQQT
		scope.state = DoneState
		return nil
	}

	if scope.lastToken == SQT && r == '\'' {
		scope.lastToken = SQQT
		return nil
	}

	if r == '\'' {
		scope.lastToken = SQT
		return nil
	}

	if scope.lastToken == SQT {
		state.Buf.WriteRune('\'')
	}

	if scope.lastToken == SQQT {
		state.Buf.WriteRune('\'')
		state.Buf.WriteRune('\'')
	}

	scope.lastToken = OTHERT
	state.Buf.WriteString(toJSONString(r))
	return nil
}

func PrefixNumber(ranges []*unicode.RangeTable, numberType NumberType) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state != InitState && unicode.IsSpace(r) {
			return parseError(state, `empty number`)
		}
		scope.state = InitState

		if unicode.IsSpace(r) || r == ']' || r == '}' || r == ',' {
			if scope.lastToken != DIGITT {
				return parseError(state, `invalid character at number end`)
			}
			state.PopScope()
			state.Buf.WriteString(strconv.FormatInt(scope.counter, 10))
			return ErrDontAdvance
		}

		if r == '_' {
			if scope.lastToken != DIGITT {
				return parseError(state, `invalid character after underscore in number`)
			}
			scope.lastToken = UNDERT
			return nil
		}

		if !unicode.IsOneOf(ranges, r) {
			return parseError(state, `invalid character in number`)
		}

		scope.lastToken = DIGITT

		var err error
		scope.counter, err = addNumber(r, scope.counter, numberType)
		if err != nil {
			return parseError(state, `addNumber failed, invalid character in number`)
		}

		return nil
	}
}

func floatDispatchSign(r rune, state *State, scope *Scope) (bool, error) {
	if r == '-' || r == '+' {
		if scope.state != OtherState &&
			scope.lastToken != EXPT {
			return false, parseError(state, `invalid sign in float`)

		}

		if scope.lastToken == SIGNT {
			return false, parseError(state, `invalid sign in float`)
		}
		scope.lastToken = SIGNT
		return true, nil
	}
	return false, nil
}

func floatDispatchUnderscore(r rune, state *State, scope *Scope) (bool, error) {
	if r == '_' {
		if scope.lastToken != DIGITT {
			return false, parseError(state, `invalid '_' in float`)
		}
		scope.counter++
		scope.lastToken = UNDERT
		return true, nil
	}
	return false, nil
}

func floatDispatchExp(r rune, state *State, scope *Scope) (bool, error) {
	if r == 'e' || r == 'E' {
		if scope.lastToken != DIGITT {
			return false, parseError(state, `invalid 'e' in float`)
		}
		scope.counter++
		scope.lastToken = EXPT
		state.Buf.WriteRune(r)
		scope.state = AfterExpState
		return true, nil
	}
	return false, nil
}

func floatDispatchDigit(r rune, state *State, scope *Scope) error {
	if !unicode.IsOneOf(digitRanges, r) {
		return parseError(state, `invalid digit in float`)
	}
	scope.lastToken = DIGITT
	scope.counter++
	state.Buf.WriteRune(r)
	return nil
}

func Float(firstState ScopeState, firstToken Token, counter int) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.counter == 0 {
			scope.state = firstState
			scope.lastToken = firstToken
			scope.counter = int64(counter)
		}

		if unicode.IsSpace(r) && scope.lastToken == DIGITT {
			state.PopScope()
			return ErrDontAdvance
		}

		if (r == ']' || r == '}' || r == ',') && scope.lastToken == DIGITT {
			state.PopScope()
			return ErrDontAdvance
		}

		if unicode.IsSpace(r) {
			return parseError(state, `invalid float ending`)
		}

		switch scope.state {
		case OtherState:

			ok, err := floatDispatchSign(r, state, scope)
			if ok || err != nil {
				if r == '-' {
					state.Buf.WriteRune(r)
				}
				return err
			}

			if r == '0' && (scope.lastToken == OTHERT || scope.lastToken == SIGNT) {
				state.Buf.WriteRune('0')
				scope.counter++
				scope.lastToken = DIGITT
				scope.state = AfterInitialZeroState
				return nil
			}

			if r == '.' {
				if scope.lastToken != DIGITT {
					return parseError(state, `invalid '.' in float`)
				}
				scope.counter++
				scope.lastToken = DOTT
				state.Buf.WriteRune('.')
				scope.state = AfterDotState
				return nil
			}

			ok, err = floatDispatchUnderscore(r, state, scope)
			if ok || err != nil {
				return err
			}

			ok, err = floatDispatchExp(r, state, scope)
			if ok || err != nil {
				return err
			}

			return floatDispatchDigit(r, state, scope)

		case AfterInitialZeroState:

			if r == '.' {
				scope.lastToken = DOTT
				state.Buf.WriteRune('.')
				scope.state = AfterDotState
				return nil
			}

			ok, err := floatDispatchExp(r, state, scope)
			if ok || err != nil {
				return err
			}

			return parseError(state, `invalid float character after zero`)

		case AfterDotState:

			ok, err := floatDispatchUnderscore(r, state, scope)
			if ok || err != nil {
				return err
			}

			ok, err = floatDispatchExp(r, state, scope)
			if ok || err != nil {
				return err
			}

			return floatDispatchDigit(r, state, scope)

		case AfterExpState:

			ok, err := floatDispatchSign(r, state, scope)
			if ok || err != nil {
				state.Buf.WriteRune(r)
				return err
			}

			ok, err = floatDispatchUnderscore(r, state, scope)
			if ok || err != nil {
				return err
			}

			return floatDispatchDigit(r, state, scope)
		}

		return nil
	}
}

func timeVerfiyHours(val []rune) error {

	if len(val) != 2 {
		return fmt.Errorf(`hours wrong lenth`)
	}

	v, err := strconv.ParseInt(string(val), 10, 8)
	if err != nil {
		return errors.Wrap(err, `verifyHours strconv.ParseInt failed`)
	}

	if v > 23 {
		return fmt.Errorf(`hours invalid`)
	}
	return nil
}

func timeVerfiy60(val []rune) error {

	if len(val) != 2 {
		return fmt.Errorf(`hours wrong lenth`)
	}

	v, err := strconv.ParseInt(string(val), 10, 8)
	if err != nil {
		return errors.Wrap(err, `verifyHours strconv.ParseInt failed`)
	}

	if v > 59 {
		return fmt.Errorf(`hours invalid`)
	}
	return nil
}

func Time(offset int, val []rune, uptoMinutes bool) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.counter == 0 {
			scope.counter += int64(offset)
			state.data = val
			state.Buf.WriteString(string(val))
		}

		if scope.counter > 8 {
			if !unicode.IsOneOf(digitRanges, r) {
				state.PopScope()
				return ErrDontAdvance
			}
		}

		if scope.counter == 8 {
			if r != '.' {
				state.PopScope()
				return ErrDontAdvance
			}
		}

		if scope.counter < 2 ||
			(scope.counter > 2 && scope.counter < 5) ||
			scope.counter > 5 && scope.counter < 8 {
			if !unicode.IsOneOf(digitRanges, r) {
				return parseError(state, `invalid digit in time`)
			}

			state.data = append(state.data, r)
		}

		if scope.counter == 5 {

			if len(state.data) != 4 {
				return parseError(state, `invalid time, hour and minutes`)
			}

			err := timeVerfiyHours(state.data[:2])
			if err != nil {
				return parseError(state, `invalid time, hours invalid`)
			}
			err = timeVerfiy60(state.data[2:])
			if err != nil {
				return parseError(state, `invalid time, minutes invalid`)
			}
			state.data = state.data[0:0]
		}

		if uptoMinutes && scope.counter == 5 {
			state.PopScope()
			return ErrDontAdvance
		}

		state.Buf.WriteRune(r)

		if scope.counter == 2 || scope.counter == 5 {
			if r != ':' {
				return parseError(state, `invalid character in time`)
			}
		}

		if scope.counter == 8 {
			if len(state.data) != 2 {
				return parseError(state, `invalid time, seconds invalid`)
			}

		}

		scope.counter++
		return nil
	}
}

func Date(offset int, val []rune) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state == AfterTState {

			if r == 'Z' {
				state.PopScope()
				state.Buf.WriteString(`Z`)
				return nil
			}

			if r == '-' || r == '+' {
				state.Buf.WriteRune(r)
				state.PopScope()
				state.PushScope(Time(0, nil, true), OtherType, nil)
				return nil
			}

			state.PopScope()
			return ErrDontAdvance
		}

		if scope.state == InitState {

			if r == ' ' || r == 'T' {
				state.Buf.WriteRune(r)
				scope.state = AfterTState
				state.PushScope(Time(0, nil, false), OtherType, scope)
				return nil
			}

			state.PopScope()
			return ErrDontAdvance
		}

		if scope.counter == 0 {
			scope.counter += int64(offset)
			state.data = val
			state.Buf.WriteString(string(val))
		}

		if scope.counter < 4 {
			if !unicode.IsOneOf(digitRanges, r) {
				return parseError(state, `invalid digit in date year`)
			}
			state.data = append(state.data, r)
		}

		if (scope.counter > 4 && scope.counter < 7) ||
			(scope.counter > 7 && scope.counter < 10) {
			if !unicode.IsOneOf(digitRanges, r) {
				return parseError(state, `invalid digit in date`)
			}
			state.data = append(state.data, r)
		}

		state.Buf.WriteRune(r)

		if scope.counter == 4 {
			if r != '-' {
				return parseError(state, `invalid character in year`)
			}

		}

		if scope.counter == 7 {
			if r != '-' {
				return parseError(state, `invalid character in date day`)
			}
		}

		scope.counter++

		if scope.counter == 10 {

			if len(state.data) != 8 {
				return parseError(state, `invalid date, month or day`)
			}

			month, err := strconv.ParseInt(string(state.data[4:6]), 10, 8)
			if err != nil {
				return parseError(state, `invalid digit in dates month`)
			}

			if month < 1 || month > 12 {
				return parseError(state, `invalid month in date`)
			}

			day, err := strconv.ParseInt(string(state.data[6:8]), 10, 8)
			if err != nil {
				return parseError(state, `invalid month in date`)
			}

			if day < 1 || int(day) > daysInMonth[month-1] {
				return parseError(state, `invalid number of days in month`)
			}

			scope.state = InitState
			return nil
		}
		return nil
	}
}

func inlineTableDispatchKeyValue(r rune, defs Defs, state *State, scope *Scope) error {

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.lastToken = OTHERT
		scope.state = AfterValueState
		state.PushScope(KeyValue(
			func(key []string, v Var) bool {
				return defs.Define(key, nil, v)
			}, defs.keyFilter.Push), OtherType, scope)
		return ErrDontAdvance
	}
	return parseError(state, `inline table could not dispatch key`)
}

func InlineTable() ParseFunc {

	defs := MakeDefs()
	return func(r rune, state *State, scope *Scope) error {

		if r == '\n' {
			return parseError(state, `inline table contains \n`)
		}

		if unicode.IsSpace(r) {
			return nil
		}

		if (scope.state == OtherState || scope.state == AfterValueState) &&
			r == '}' {
			state.PopScope()

			if scope.lastToken == COMT {
				return parseError(state, `inline table invalid comma at end`)
			}

			defs.keyFilter.Close(state.Buf)
			state.Buf.WriteRune('}')
			return nil
		}

		if scope.state == AfterValueState {

			if r == ',' && scope.lastToken != COMT {
				scope.lastToken = COMT
				return nil
			}

			if scope.lastToken != COMT {
				return parseError(state, `inline table comma not found`)
			}
		}

		return inlineTableDispatchKeyValue(r, defs, state, scope)
	}
}

func InlineArray(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) {
		return nil
	}

	if r == ']' {
		state.PopScope()
		state.Buf.WriteRune(']')
		return nil
	}

	if scope.state == AfterValueState {

		if r == ',' && scope.lastToken != COMT {
			scope.lastToken = COMT
			return nil
		}

		if scope.lastToken != COMT {
			return parseError(state, `inline table comma not found`)
		}
		state.Buf.WriteRune(',')
	}
	scope.lastToken = OTHERT
	scope.state = AfterValueState
	state.PushScope(Value, OtherType, scope)
	return ErrDontAdvance
}

func LiteralValue(value []rune) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if int(scope.counter) < len(value) {
			if value[scope.counter] != r {
				return parseError(state, `invalid literal value`)
			}
		}

		scope.counter++
		if int(scope.counter) >= len(value) {
			state.PopScope()
			return nil
		}
		return nil
	}
}

func DateOrTime(hasZeroPrefix bool) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state == AfterValueState {

			state.Buf.WriteString(`"`)
			state.PopScope()
			return ErrDontAdvance
		}

		if scope.counter == 0 && hasZeroPrefix {
			state.data = append(state.data, '0')
			scope.counter++
		}

		if r == ':' && len(state.data) == 2 {

			state.Buf.WriteString(`"`)
			scope.state = AfterValueState
			state.PushScope(Time(2, state.data, false), OtherType, scope)
			return ErrDontAdvance
		}

		if r == '-' && len(state.data) == 4 {

			state.Buf.WriteString(`"`)
			scope.state = AfterValueState
			state.PushScope(Date(4, state.data), OtherType, scope)
			return ErrDontAdvance
		}

		if len(state.data) == 4 {
			return parseError(state, `dateOrTime unexpected character`)
		}

		if !unicode.IsOneOf(digitRanges, r) {
			return parseError(state, `digit expected`)
		}

		state.data = append(state.data, r)
		return nil
	}
}

func SignedNumber(r rune, state *State, scope *Scope) error {

	if scope.state == OtherState {
		if r == '-' || r == '+' {
			state.data = append(state.data, r)
			scope.state = InitState
			return nil
		}
	}

	if r == 'n' {

		state.Buf.WriteRune('"')
		if len(state.data) == 1 {
			state.Buf.WriteRune(state.data[0])
		}
		state.Buf.WriteString(`nan"`)

		state.PopScope()
		state.PushScope(LiteralValue(nanRunes), OtherType, nil)
		return ErrDontAdvance
	}

	if r == 'i' {

		state.Buf.WriteRune('"')
		if len(state.data) == 1 {
			state.Buf.WriteRune(state.data[0])
		}
		state.Buf.WriteString(`inf"`)

		state.PopScope()
		state.PushScope(LiteralValue(infRunes), OtherType, nil)
		return ErrDontAdvance
	}

	firstToken := OTHERT
	if len(state.data) == 1 {
		if state.data[0] == '-' {
			state.Buf.WriteRune('-')
		}
		firstToken = SIGNT
	}

	state.PopScope()
	state.PushScope(Float(OtherState, firstToken, len(state.data)), OtherType, nil)
	return ErrDontAdvance
}

func NumberDateOrTime(r rune, state *State, scope *Scope) error {

	if scope.state == AfterValueState {

		state.Buf.WriteString(`"`)
		state.PopScope()
		return ErrDontAdvance
	}

	if len(state.data) == 0 && (r == '+' || r == '-' || r == 'n' || r == 'i') {

		state.PopScope()
		state.PushScope(SignedNumber, OtherType, nil)
		return ErrDontAdvance
	}

	if r == '}' ||
		r == ']' ||
		r == ',' ||
		r == '_' ||
		unicode.IsSpace(r) ||
		scope.counter >= 3 ||
		r == '.' || r == 'e' || r == 'E' {

		if len(state.data) == 0 {
			return parseError(state, `invalid character in number`)
		}

		state.Buf.WriteString(string(state.data))
		state.PopScope()
		state.PushScope(Float(OtherState, DIGITT, len(state.data)), OtherType, nil)
		return ErrDontAdvance
	}

	if r == ':' && len(state.data) == 2 {

		state.Buf.WriteString(`"`)
		scope.state = AfterValueState
		state.PushScope(Time(2, state.data, false), OtherType, scope)
		return ErrDontAdvance
	}

	if r == '-' && len(state.data) == 4 {

		scope.state = AfterValueState
		state.Buf.WriteString(`"`)
		state.PushScope(Date(4, state.data), OtherType, scope)
		return ErrDontAdvance
	}

	// TODO check the zero case
	if !unicode.IsOneOf(digitRanges, r) {
		return parseError(state, `digit expected 2`)
	}

	state.data = append(state.data, r)

	return nil
}

func Zero(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) || r == ',' || r == ']' || r == '}' {
		state.Buf.WriteString(`0`)
		state.PopScope()
		return ErrDontAdvance
	}

	if r == 'x' {
		state.PopScope()
		state.PushScope(PrefixNumber(hexRanges, HexNumberType), OtherType, nil)
		return nil
	}

	if r == 'o' {
		state.PopScope()
		state.PushScope(PrefixNumber(octalRanges, OctalNumberType), OtherType, nil)
		return nil
	}

	if r == 'b' {
		state.PopScope()
		state.PushScope(PrefixNumber(binRanges, BinNumberType), OtherType, nil)
		return nil
	}

	if r == 'e' || r == 'E' {
		state.PopScope()
		state.Buf.WriteRune('0')
		state.Buf.WriteRune(r)
		state.PushScope(Float(AfterExpState, OTHERT, 2), OtherType, nil)
		return nil
	}

	if unicode.IsOneOf(digitRanges, r) {
		state.PopScope()
		state.PushScope(DateOrTime(true), OtherType, nil)
		return ErrDontAdvance
	}

	if r == '.' {
		state.PopScope()
		state.Buf.WriteString("0.")
		state.PushScope(Float(AfterDotState, OTHERT, 2), OtherType, nil)
		return nil
	}

	return parseError(state, `invalid character after zero`)
}

func Value(r rune, state *State, scope *Scope) error {

	if scope.lastToken == OTHERT && r == '"' {
		scope.lastToken = QT
		scope.scopeType = StringType
		return nil
	}

	if scope.lastToken == QT && r != '"' {
		state.PopScope()
		state.PushScope(QuotedString, StringType, nil)
		state.Buf.WriteRune('"')
		return ErrDontAdvance
	}

	if scope.lastToken == QT && r == '"' {
		scope.lastToken = QQT
		return nil
	}

	if scope.lastToken == QQT && r != '"' {
		state.PopScope()
		state.Buf.WriteString(`""`)
		return ErrDontAdvance
	}

	if scope.lastToken == QQT && r == '"' {
		state.PopScope()
		state.PushScope(TrippleQuotedString, StringType, nil)
		state.Buf.WriteRune('"')
		return nil
	}

	if scope.lastToken == OTHERT && r == '\'' {
		scope.lastToken = SQT
		scope.scopeType = StringType
		return nil
	}

	if scope.lastToken == SQT && r != '\'' {
		state.PopScope()
		state.PushScope(LiteralString, StringType, nil)
		state.Buf.WriteRune('"')
		return ErrDontAdvance
	}

	if scope.lastToken == SQT && r == '\'' {
		scope.lastToken = SQQT
		return nil
	}

	if scope.lastToken == SQQT && r != '\'' {
		state.PopScope()
		state.Buf.WriteString(`""`)
		return ErrDontAdvance
	}

	if scope.lastToken == SQQT && r == '\'' {
		state.PopScope()
		state.PushScope(MultiLineLiteralString, StringType, nil)
		state.Buf.WriteRune('"')
		return nil
	}

	if unicode.IsSpace(r) {
		return nil
	}

	if r == 't' {
		state.PopScope()
		state.PushScope(LiteralValue(trueRunes), OtherType, nil)
		state.Buf.WriteString(`true`)
		return ErrDontAdvance
	}

	if r == 'f' {
		state.PopScope()
		state.PushScope(LiteralValue(falseRunes), OtherType, nil)
		state.Buf.WriteString(`false`)
		return ErrDontAdvance
	}

	if r == '{' {
		state.PopScope()
		state.PushScope(InlineTable(), OtherType, nil)
		state.Buf.WriteRune('{')
		return nil
	}

	if r == '[' {
		state.PopScope()
		state.PushScope(InlineArray, OtherType, nil)
		state.Buf.WriteRune('[')
		return nil
	}

	if r == '0' {
		state.PopScope()
		state.PushScope(Zero, OtherType, nil)
		return nil
	}

	state.PopScope()
	state.PushScope(NumberDateOrTime, OtherType, nil)
	return ErrDontAdvance
}

func Key(r rune, state *State, scope *Scope) error {

	if unicode.IsSpace(r) && r != '\n' {
		if scope.lastToken != DOTT {
			scope.lastToken = SPACET
		}
		return nil
	}

	if scope.state != OtherState && (unicode.IsSpace(r) || r == '=' || r == ']') {

		if scope.lastToken == DOTT {
			return parseError(state, `invalid '.' at the end of the key`)
		}
		state.PopScope()
		return ErrDontAdvance
	}

	if (scope.state == OtherState || scope.lastToken == DOTT) && r == '.' {
		return parseError(state, `invalid '.' in key`)
	}

	st := scope.state
	if scope.state == OtherState {
		scope.state = InitState
	}

	if scope.state == AfterQuoteState {
		if r != '.' {
			return parseError(state, `invalid character in key after quote`)
		}
		scope.lastToken = DOTT
		scope.state = InitState
	}

	if r == '.' {
		scope.lastToken = DOTT
		state.keyData = append(state.keyData, '\n')
		return nil
	}

	if r == '"' {
		if scope.lastToken != DOTT && st != OtherState {
			return parseError(state, `invalid character before '"' in key`)
		}
		scope.lastToken = OTHERT
		scope.state = AfterQuoteState
		state.PushScope(QuotedString, KeyType, scope)
		return nil
	}

	if r == '\'' {
		if scope.lastToken != DOTT && st != OtherState {
			return parseError(state, `invalid character before '\'' in key`)
		}
		scope.lastToken = OTHERT
		scope.state = AfterQuoteState
		state.PushScope(LiteralString, KeyType, scope)
		return nil
	}

	if !unicode.IsOneOf(bareRanges, r) {
		return parseError(state, `invalid character in key`)
	}

	if scope.lastToken == SPACET {
		return parseError(state, `invalid space in key`)
	}

	scope.lastToken = OTHERT
	state.keyData = append(state.keyData, r)
	return nil
}

func KeyValue(defineFunc DefineFunc, pushFilter KeyFilterPushFunc) ParseFunc {
	return func(r rune, state *State, scope *Scope) error {

		if scope.state == AfterValueState {

			if r == EOF || r == '\n' || r == ',' || r == '}' || r == ']' {

				state.PopScope()
				return ErrDontAdvance
			}

			if !unicode.IsSpace(r) {
				return parseError(state, `invalid character after value`)
			}
			return nil
		}
		if scope.state == AfterKeyState {

			if r == '\n' {
				return parseError(state, `invalid '\n' after key`)
			}

			if scope.lastToken == OTHERT && unicode.IsSpace(r) {
				return nil
			}

			if r == '=' {

				scope.key = state.ExtractKeys()
				ok := defineFunc(scope.key, BasicVar)
				if !ok {
					return parseError(state, `attempt to redefine a key`)
				}

				scope.state = AfterValueState

				pushFilter(scope.key, BasicVar, state.Buf)

				state.Buf.WriteString(`"` + scope.key[len(scope.key)-1] + `":`)

				state.PushScope(Value, OtherType, scope)
				return nil
			}

			return parseError(state, `invalid key`)
		}

		if !unicode.IsSpace(r) {
			scope.state = AfterKeyState
			state.PushScope(Key, KeyType, scope)
			return ErrDontAdvance
		}
		return nil
	}
}

func Table(r rune, state *State, scope *Scope) error {

	if r == '[' || r == EOF {
		state.PopScope()
		return ErrDontAdvance
	}

	if scope.state == AfterValueState {

		if r == '\n' {
			scope.state = AfterFirstValueState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid value in table`)
		}
		return nil
	}

	if scope.state == AfterTableState || scope.state == AfterFirstValueState {

		if !unicode.IsSpace(r) {
			scope.state = AfterValueState
			state.PushScope(
				KeyValue(
					BaseKeyDefs(scope.key, scope.key, state.defs),
					BaseKeyFilterPushFunc(scope.key, state.defs.keyFilter.Push),
				),
				OtherType, scope)

			return ErrDontAdvance
		}
		return nil
	}

	if r == '\n' {
		if scope.lastToken != CBT {
			return parseError(state, `table end invalid`)
		}
		scope.key = state.ExtractKeys()

		ok := state.defs.Define(scope.key, nil, TableVar)
		if !ok {
			return parseError(state, `table attempt to redefine a key`)
		}
		state.defs.keyFilter.Push(scope.key, TableVar, state.Buf)

		scope.state = AfterTableState
		return nil
	}

	if unicode.IsSpace(r) {
		return nil
	}

	if scope.state == AfterKeyState {

		if scope.lastToken != CBT && r == ']' {
			scope.lastToken = CBT
			return nil
		}

		return parseError(state, `invalid character at table end`)
	}

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.state = AfterKeyState
		state.PushScope(Key, KeyType, scope)
		return ErrDontAdvance
	}
	return parseError(state, `invalid character at table start`)
}

func Array(r rune, state *State, scope *Scope) error {

	if r == '[' || r == EOF {
		state.PopScope()
		return ErrDontAdvance
	}

	if scope.state == AfterValueState {

		if r == '\n' {
			scope.state = AfterFirstValueState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid value in array`)
		}
		return nil
	}

	if scope.state == AfterArrayState || scope.state == AfterFirstValueState {
		if !unicode.IsSpace(r) {
			scope.state = AfterValueState
			state.PushScope(
				KeyValue(
					BaseKeyDefs(scope.key, nil, state.defs),
					BaseKeyFilterPushFunc(scope.key, state.defs.keyFilter.Push),
				),
				OtherType,
				scope,
			)
			return ErrDontAdvance
		}
		return nil
	}

	if r == '\n' {
		if scope.lastToken != CBBT {
			return parseError(state, `array end invalid`)
		}
		scope.key = state.ExtractKeys()
		ok := state.defs.Define(scope.key, nil, ArrayVar)
		if !ok {
			return parseError(state, `array attempt to redefine a key`)
		}
		state.defs.keyFilter.Push(scope.key, ArrayVar, state.Buf)

		scope.state = AfterArrayState
		return nil
	}

	if unicode.IsSpace(r) {
		if scope.lastToken == CBT {
			scope.lastToken = CBTSPACE
		}
		return nil
	}

	if scope.state == AfterKeyState {

		if scope.lastToken == CBT && r == ']' {
			scope.lastToken = CBBT
			return nil
		}

		if scope.lastToken == CBTSPACE && r == ']' {
			return parseError(state, `array invalid right brace`)
		}

		if r == ']' {
			scope.lastToken = CBT
			return nil
		}

		return parseError(state, `invalid character at array end`)
	}

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.state = AfterKeyState
		state.PushScope(Key, KeyType, scope)
		return ErrDontAdvance
	}
	return parseError(state, `invalid character at array start`)
}

func Top(r rune, state *State, scope *Scope) error {

	if r == EOF && scope.lastToken != OTHERT {
		return parseError(state, `invalid end of file`)
	}
	if r == EOF {
		state.PopScope()
		return nil
	}

	if scope.state == AfterValueState {

		if r == '[' {
			scope.state = AfterFirstValueState
			return ErrDontAdvance
		}

		if r == '\n' {
			scope.state = AfterFirstValueState
			return nil
		}

		if !unicode.IsSpace(r) {
			return parseError(state, `invalid character after value`)
		}

		return nil
	}

	if unicode.IsSpace(r) {
		if scope.lastToken == CBT {
			scope.lastToken = CBTSPACE
		}
		return nil
	}

	if scope.lastToken == CBTSPACE && r == '[' {
		return parseError(state, `invalid left brace`)
	}

	if scope.lastToken == CBT && r == '[' {
		scope.lastToken = OTHERT
		state.PushScope(Array, OtherType, scope)
		return nil
	}

	if scope.lastToken == CBT || scope.lastToken == CBTSPACE {
		scope.lastToken = OTHERT
		state.PushScope(Table, OtherType, scope)
		return ErrDontAdvance
	}

	if r == '[' {
		scope.lastToken = CBT
		return nil
	}

	if unicode.IsOneOf(bareRanges, r) || r == '"' || r == '\'' {
		scope.lastToken = OTHERT
		scope.state = AfterValueState
		state.PushScope(KeyValue(
			func(key []string, v Var) bool {
				return state.defs.Define(key, nil, v)
			}, state.defs.keyFilter.Push), OtherType, scope)
		return ErrDontAdvance
	}

	return parseError(state, fmt.Sprintf(`invalid character (%v)`, r))
}

func parseError(s *State, msg string) error {
	return fmt.Errorf("position (%v:%v) msg: %v", s.line, s.position, msg)
}

func isOneOf(r rune, runes []rune) bool {

	for _, rn := range runes {
		if r == rn {
			return true
		}
	}
	return false
}
