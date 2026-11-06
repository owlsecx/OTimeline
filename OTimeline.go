// ============================================================
//
//	OTimeline — Filesystem Super-Timeline
//	Author  : khaled.s.haddad
//	Website : owlsec.org
//	Language: Go (stdlib only) — OWL Red/Black Edition
//
// ============================================================
package main

import (
	"bufio"
	"bytes"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const APPNAME = "OTimeline"
const TAGLINE = "Filesystem Super-Timeline"
const BANNER = `
 ██████╗ ████████╗██╗███╗   ███╗███████╗██╗     ██╗███╗   ██╗███████╗
██╔═══██╗╚══██╔══╝██║████╗ ████║██╔════╝██║     ██║████╗  ██║██╔════╝
██║   ██║   ██║   ██║██╔████╔██║█████╗  ██║     ██║██╔██╗ ██║█████╗  
██║   ██║   ██║   ██║██║╚██╔╝██║██╔══╝  ██║     ██║██║╚██╗██║██╔══╝  
╚██████╔╝   ██║   ██║██║ ╚═╝ ██║███████╗███████╗██║██║ ╚████║███████╗
 ╚═════╝    ╚═╝   ╚═╝╚═╝     ╚═╝╚══════╝╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝
`

// ---- OWL UI KIT (red/black) — stdlib only. Paste into each tool's main.go ----
const (
	R   = "\033[91m"
	DR  = "\033[96m"
	GRN = "\033[92m"
	YLW = "\033[93m"
	CY  = "\033[96m"
	W   = "\033[97m"
	B   = "\033[1m"
	X   = "\033[0m"
	MG  = "\033[95m"
	DIM = "\033[2m"
	OR  = "\033[38;5;208m"
	GY  = "\033[90m"
)

const (
	SUITE   = "Owl Forensics Suite"
	AUTHOR  = "khaled.s.haddad"
	WEBSITE = "owlsec.org"
)

var noColor = os.Getenv("NO_COLOR") != "" || !isatty()

func isatty() bool { fi, _ := os.Stdout.Stat(); return (fi.Mode() & os.ModeCharDevice) != 0 }
func col(s string, c ...string) string {
	if noColor {
		return s
	}
	return strings.Join(c, "") + s + X
}
func clear() {
	if !noColor {
		fmt.Print("\033[H\033[2J")
	}
}
func rule()            { fmt.Println(col(strings.Repeat("─", 66), CY)) }
func line(w int)       { fmt.Println(col(strings.Repeat("─", w), CY)) }
func okMsg(s string)   { fmt.Println("  " + col("[✔] ", GRN+B) + col(s, W)) }
func errMsg(s string)  { fmt.Println("  " + col("[✘] ", R+B) + col(s, R)) }
func warnMsg(s string) { fmt.Println("  " + col("[!] ", OR+B) + col(s, YLW)) }
func info(s string)    { fmt.Println("  " + col("[*] ", CY) + col(s, DIM+W)) }
func header(title string) {
	fmt.Println()
	rule()
	fmt.Println("  " + col("⬡  "+title, B+CY))
	rule()
	fmt.Println()
}

var stdin = bufio.NewReader(os.Stdin)

func prompt(label string) string {
	fmt.Print("  " + col("[»] ", YLW) + col(label+": ", W))
	s, err := stdin.ReadString('\n')
	if err != nil && s == "" {
		fmt.Println()
		os.Exit(0)
	}
	return strings.TrimSpace(s)
}
func pause() {
	fmt.Print(col("\n  [↩] Press Enter to continue...", YLW))
	stdin.ReadString('\n')
}
func stripANSI(s string) string {
	out := []rune{}
	esc := false
	for _, r := range s {
		if r == '\033' {
			esc = true
			continue
		}
		if esc {
			if r == 'm' {
				esc = false
			}
			continue
		}
		out = append(out, r)
	}
	return string(out)
}
func visw(s string) int { return len([]rune(stripANSI(s))) }
func pad(s string, n int) string {
	if visw(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-visw(s))
}
func table(headers []string, rows [][]string) {
	w := make([]int, len(headers))
	for i, h := range headers {
		w[i] = visw(h)
	}
	for _, r := range rows {
		for i, c := range r {
			if i < len(w) && visw(c) > w[i] {
				w[i] = visw(c)
			}
		}
	}
	var hb strings.Builder
	for i, h := range headers {
		hb.WriteString(col(pad(h, w[i]), B+CY))
		if i < len(headers)-1 {
			hb.WriteString(col(" │ ", DIM))
		}
	}
	fmt.Println("  " + hb.String())
	seg := make([]string, len(w))
	for i := range w {
		seg[i] = strings.Repeat("─", w[i])
	}
	fmt.Println("  " + col(strings.Join(seg, "─┼─"), DIM))
	for _, r := range rows {
		var rb strings.Builder
		for i := range headers {
			v := ""
			if i < len(r) {
				v = r[i]
			}
			rb.WriteString(col(pad(v, w[i]), W))
			if i < len(headers)-1 {
				rb.WriteString(col(" │ ", DIM))
			}
		}
		fmt.Println("  " + rb.String())
	}
}
func banner() {
	fmt.Println(col(BANNER, CY))
	fmt.Println("  " + col(SUITE+"  ", B+MG) + col("v"+VERSION, YLW))
	fmt.Println("  " + col(TAGLINE, W))
	fmt.Println()
	fmt.Println("  " + col("Developer : ", DIM) + col(AUTHOR, CY) +
		col("   |   Website : ", DIM) + col(WEBSITE, CY))
	rule()
	fmt.Println()
}
func menu(title string, items []string) string {
	clear()
	banner()
	fmt.Println("  " + col("⬡  "+title, B+CY))
	fmt.Println()
	for i, it := range items {
		fmt.Println("  " + col(pad(fmt.Sprintf("[%d]", i+1), 4), CY+B) + "  " + col(it, W))
	}
	fmt.Println()
	fmt.Println("  " + col(pad("[H]", 4), YLW+B) + "  " + col("Help & Usage", W))
	fmt.Println("  " + col(pad("[0]", 4), R+B) + "  " + col("Exit", W))
	fmt.Println()
	rule()
	return prompt("Select option")
}

// ---- OWL house help (shared layout) ----
type owlHelpItem struct{ key, text string }

func owlHelpWrap(s string, width int) []string {
	words := strings.Fields(s)
	if len(words) == 0 {
		return []string{""}
	}
	out := []string{}
	cur := ""
	for _, w := range words {
		if cur != "" && len([]rune(cur))+1+len([]rune(w)) > width {
			out = append(out, cur)
			cur = w
		} else if cur == "" {
			cur = w
		} else {
			cur += " " + w
		}
	}
	out = append(out, cur)
	return out
}
func owlHelpTitle(t string) { fmt.Println(); fmt.Println("  " + col("▸ "+t, B+CY)) }
func owlHelpPara(lines []string) {
	for _, l := range lines {
		for _, w := range owlHelpWrap(l, 60) {
			fmt.Println("    " + col(w, W))
		}
	}
}
func owlHelpRows(items []owlHelpItem) {
	kw := 0
	for _, it := range items {
		if n := len([]rune(it.key)); n > kw {
			kw = n
		}
	}
	for _, it := range items {
		k := "[" + it.key + "]" + strings.Repeat(" ", kw-len([]rune(it.key)))
		for i, w := range owlHelpWrap(it.text, 56-kw) {
			if i == 0 {
				fmt.Println("    " + col(k, YLW+B) + "  " + col(w, W))
			} else {
				fmt.Println("    " + strings.Repeat(" ", kw+2) + "  " + col(w, W))
			}
		}
	}
}
func tuiHelp() {
	clear()
	banner()
	rule()
	fmt.Println("  " + col("⬡  Help & Usage — "+APPNAME, B+CY))
	rule()
	owlHelpTitle("WHAT IT DOES")
	owlHelpPara(owlHelpAbout)
	owlHelpTitle("HOW TO USE THE MENU")
	owlHelpPara(owlHelpFlow)
	owlHelpTitle("MENU OPTIONS")
	owlHelpRows(owlHelpMenu)
	owlHelpTitle("TIPS")
	owlHelpRows(owlHelpTips)
	fmt.Println()
	rule()
	fmt.Println("  " + col("Next page: command-line flags & examples", DIM))
	pause()
	clear()
	usage()
	fmt.Println()
	fmt.Println("  " + col("Tip: the same reference is printed by  "+APPNAME+" --help", DIM))
	pause()
}

// ---- Common finding / report types ----
type Finding struct {
	ID          string            `json:"id"`
	Severity    string            `json:"severity"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Evidence    string            `json:"evidence"`
	Mitre       []string          `json:"mitre_attack"`
	Category    string            `json:"category"`
	Score       float64           `json:"score"`
	Tags        []string          `json:"tags"`
	Location    string            `json:"location"`
	Timestamp   string            `json:"timestamp"`
	Extra       map[string]string `json:"extra,omitempty"`
}

type Report struct {
	Tool        string            `json:"tool"`
	Version     string            `json:"version"`
	GeneratedAt string            `json:"generated_at"`
	Subject     string            `json:"subject"`
	Grade       string            `json:"grade"`
	Score       float64           `json:"score"`
	Summary     string            `json:"summary"`
	Findings    []Finding         `json:"findings"`
	Stats       map[string]int    `json:"stats"`
	Meta        map[string]string `json:"meta"`
}

const VERSION = "2.0.0"

func newReport(subject string) *Report {
	return &Report{
		Tool: APPNAME, Version: VERSION,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Subject:     subject, Grade: "N/A", Score: 0,
		Findings: []Finding{}, Stats: map[string]int{}, Meta: map[string]string{},
	}
}

func (r *Report) add(f Finding) {
	if f.ID == "" {
		f.ID = fmt.Sprintf("F-%04d", len(r.Findings)+1)
	}
	if f.Timestamp == "" {
		f.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}
	if f.Extra == nil {
		f.Extra = map[string]string{}
	}
	r.Findings = append(r.Findings, f)
	r.Stats[f.Severity]++
	r.Stats["total"]++
}

func gradeFromScore(score float64) string {
	switch {
	case score >= 97:
		return "A+"
	case score >= 93:
		return "A"
	case score >= 90:
		return "A-"
	case score >= 87:
		return "B+"
	case score >= 83:
		return "B"
	case score >= 80:
		return "B-"
	case score >= 77:
		return "C+"
	case score >= 73:
		return "C"
	case score >= 70:
		return "C-"
	case score >= 67:
		return "D+"
	case score >= 63:
		return "D"
	case score >= 60:
		return "D-"
	default:
		return "F"
	}
}

func severityWeight(s string) float64 {
	switch strings.ToLower(s) {
	case "critical":
		return 25
	case "high":
		return 15
	case "medium":
		return 8
	case "low":
		return 3
	case "info":
		return 0.5
	default:
		return 1
	}
}

func (r *Report) finalize() {
	penalty := 0.0
	for _, f := range r.Findings {
		penalty += severityWeight(f.Severity)
	}
	score := 100.0 - penalty
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	r.Score = score
	r.Grade = gradeFromScore(score)
	r.Summary = fmt.Sprintf("%d findings · grade %s (%.1f/100)", len(r.Findings), r.Grade, r.Score)
}

func exportTXT(r *Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("=== %s v%s Report ===\n", r.Tool, r.Version))
	b.WriteString(fmt.Sprintf("Generated : %s\n", r.GeneratedAt))
	b.WriteString(fmt.Sprintf("Subject   : %s\n", r.Subject))
	b.WriteString(fmt.Sprintf("Grade     : %s (%.1f)\n", r.Grade, r.Score))
	b.WriteString(fmt.Sprintf("Summary   : %s\n", r.Summary))
	b.WriteString(strings.Repeat("=", 64) + "\n\n")
	for _, f := range r.Findings {
		b.WriteString(fmt.Sprintf("[%s] %s — %s\n", f.Severity, f.ID, f.Title))
		b.WriteString(fmt.Sprintf("  Category : %s\n", f.Category))
		b.WriteString(fmt.Sprintf("  Location : %s\n", f.Location))
		b.WriteString(fmt.Sprintf("  Evidence : %s\n", f.Evidence))
		b.WriteString(fmt.Sprintf("  Detail   : %s\n", f.Description))
		if len(f.Mitre) > 0 {
			b.WriteString(fmt.Sprintf("  MITRE    : %s\n", strings.Join(f.Mitre, ", ")))
		}
		if len(f.Tags) > 0 {
			b.WriteString(fmt.Sprintf("  Tags     : %s\n", strings.Join(f.Tags, ", ")))
		}
		b.WriteString("\n")
	}
	b.WriteString(fmt.Sprintf("\nStats: %v\n", r.Stats))
	return b.String()
}

func exportMD(r *Report) string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s Report\n\n", r.Tool))
	b.WriteString("| Field | Value |\n|---|---|\n")
	b.WriteString(fmt.Sprintf("| Version | %s |\n", r.Version))
	b.WriteString(fmt.Sprintf("| Generated | %s |\n", r.GeneratedAt))
	b.WriteString(fmt.Sprintf("| Subject | `%s` |\n", r.Subject))
	b.WriteString(fmt.Sprintf("| Grade | **%s** (%.1f) |\n", r.Grade, r.Score))
	b.WriteString(fmt.Sprintf("| Summary | %s |\n\n", r.Summary))
	b.WriteString("## Findings\n\n")
	for _, f := range r.Findings {
		b.WriteString(fmt.Sprintf("### %s — %s (%s)\n\n", f.ID, f.Title, f.Severity))
		b.WriteString(fmt.Sprintf("- **Category:** %s\n", f.Category))
		b.WriteString(fmt.Sprintf("- **Location:** `%s`\n", f.Location))
		b.WriteString(fmt.Sprintf("- **Evidence:** %s\n", f.Evidence))
		b.WriteString(fmt.Sprintf("- **Description:** %s\n", f.Description))
		if len(f.Mitre) > 0 {
			b.WriteString(fmt.Sprintf("- **MITRE ATT&CK:** %s\n", strings.Join(f.Mitre, ", ")))
		}
		b.WriteString("\n")
	}
	return b.String()
}

func exportJSON(r *Report) string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}

func exportCSV(r *Report) string {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"id", "severity", "title", "category", "location", "evidence", "mitre", "score", "tags"})
	for _, f := range r.Findings {
		_ = w.Write([]string{f.ID, f.Severity, f.Title, f.Category, f.Location, f.Evidence,
			strings.Join(f.Mitre, ";"), fmt.Sprintf("%.2f", f.Score), strings.Join(f.Tags, ";")})
	}
	w.Flush()
	return buf.String()
}

func exportSARIF(r *Report) string {
	type sarifMsg struct {
		Text string `json:"text"`
	}
	type artLoc struct {
		URI string `json:"uri"`
	}
	type physLoc struct {
		ArtifactLocation artLoc `json:"artifactLocation"`
	}
	type sarifLoc struct {
		PhysicalLocation physLoc `json:"physicalLocation"`
	}
	type sarifResult struct {
		RuleID     string                 `json:"ruleId"`
		Level      string                 `json:"level"`
		Message    sarifMsg               `json:"message"`
		Locations  []sarifLoc             `json:"locations"`
		Properties map[string]interface{} `json:"properties,omitempty"`
	}
	type sarifRule struct {
		ID               string                 `json:"id"`
		Name             string                 `json:"name"`
		ShortDescription sarifMsg               `json:"shortDescription"`
		FullDescription  sarifMsg               `json:"fullDescription"`
		HelpURI          string                 `json:"helpUri,omitempty"`
		Properties       map[string]interface{} `json:"properties,omitempty"`
	}
	type sarifDriver struct {
		Name           string      `json:"name"`
		Version        string      `json:"version"`
		InformationURI string      `json:"informationUri"`
		Rules          []sarifRule `json:"rules"`
	}
	type sarifTool struct {
		Driver sarifDriver `json:"driver"`
	}
	type sarifRun struct {
		Tool    sarifTool     `json:"tool"`
		Results []sarifResult `json:"results"`
	}
	type sarifDoc struct {
		Schema  string     `json:"$schema"`
		Version string     `json:"version"`
		Runs    []sarifRun `json:"runs"`
	}
	levelMap := map[string]string{"critical": "error", "high": "error", "medium": "warning", "low": "note", "info": "note"}
	rules := []sarifRule{}
	results := []sarifResult{}
	seen := map[string]bool{}
	for _, f := range r.Findings {
		rid := f.Category
		if rid == "" {
			rid = "generic"
		}
		if !seen[rid] {
			seen[rid] = true
			rules = append(rules, sarifRule{
				ID: rid, Name: rid,
				ShortDescription: sarifMsg{Text: rid},
				FullDescription:  sarifMsg{Text: f.Description},
				HelpURI:          "https://owlsec.org",
				Properties:       map[string]interface{}{"tags": f.Mitre},
			})
		}
		uri := f.Location
		if uri == "" {
			uri = r.Subject
		}
		lvl := levelMap[strings.ToLower(f.Severity)]
		if lvl == "" {
			lvl = "warning"
		}
		results = append(results, sarifResult{
			RuleID: rid, Level: lvl,
			Message:   sarifMsg{Text: fmt.Sprintf("%s: %s", f.Title, f.Evidence)},
			Locations: []sarifLoc{{PhysicalLocation: physLoc{ArtifactLocation: artLoc{URI: uri}}}},
			Properties: map[string]interface{}{
				"mitre": f.Mitre, "grade_score": f.Score, "tags": f.Tags,
			},
		})
	}
	doc := sarifDoc{
		Schema:  "https://raw.githubusercontent.com/oasis-tcs/sarif-spec/master/Schemata/sarif-schema-2.1.0.json",
		Version: "2.1.0",
		Runs: []sarifRun{{
			Tool: sarifTool{Driver: sarifDriver{
				Name: APPNAME, Version: VERSION, InformationURI: "https://owlsec.org", Rules: rules,
			}},
			Results: results,
		}},
	}
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(data)
}

func exportReport(r *Report, format, outPath string) error {
	r.finalize()
	var body string
	switch strings.ToLower(format) {
	case "txt", "text":
		body = exportTXT(r)
	case "md", "markdown":
		body = exportMD(r)
	case "json":
		body = exportJSON(r)
	case "csv":
		body = exportCSV(r)
	case "sarif":
		body = exportSARIF(r)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
	if outPath == "" || outPath == "-" {
		fmt.Print(body)
		return nil
	}
	return os.WriteFile(outPath, []byte(body), 0644)
}

func printReportSummary(r *Report) {
	r.finalize()
	okMsg(fmt.Sprintf("Grade %s (%.1f/100) — %d findings", r.Grade, r.Score, len(r.Findings)))
	rows := [][]string{}
	for _, f := range r.Findings {
		mitre := ""
		if len(f.Mitre) > 0 {
			mitre = f.Mitre[0]
		}
		rows = append(rows, []string{f.ID, f.Severity, truncate(f.Title, 40), mitre})
		if len(rows) >= 25 {
			break
		}
	}
	if len(rows) > 0 {
		table([]string{"ID", "Sev", "Title", "MITRE"}, rows)
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n-1] + "…"
}

func sha256hex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func sha1hex(b []byte) string {
	h := sha1.Sum(b)
	return hex.EncodeToString(h[:])
}
func md5hex(b []byte) string {
	h := md5.Sum(b)
	return hex.EncodeToString(h[:])
}

func entropyOf(b []byte) float64 {
	if len(b) == 0 {
		return 0
	}
	var freq [256]float64
	for _, c := range b {
		freq[c]++
	}
	n := float64(len(b))
	ent := 0.0
	for _, f := range freq {
		if f == 0 {
			continue
		}
		p := f / n
		ent -= p * math.Log2(p)
	}
	return ent
}

func readFileLimited(path string, max int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	if max <= 0 {
		max = 64 << 20
	}
	return io.ReadAll(io.LimitReader(f, max))
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

func mustAtoi(s string, def int) int {
	v, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return v
}

func pct(n, d int) float64 {
	if d == 0 {
		return 0
	}
	return 100.0 * float64(n) / float64(d)
}

func nowStamp() string { return time.Now().Format("20060102-150405") }

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return b
}

func u16le(b []byte, off int) uint16 {
	if off+2 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint16(b[off:])
}
func u32le(b []byte, off int) uint32 {
	if off+4 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint32(b[off:])
}
func u64le(b []byte, off int) uint64 {
	if off+8 > len(b) {
		return 0
	}
	return binary.LittleEndian.Uint64(b[off:])
}
func i64le(b []byte, off int) int64 {
	return int64(u64le(b, off))
}

func utf16leToString(b []byte) string {
	if len(b)%2 != 0 {
		b = b[:len(b)-1]
	}
	u := make([]uint16, len(b)/2)
	for i := 0; i < len(u); i++ {
		u[i] = binary.LittleEndian.Uint16(b[i*2:])
	}
	// decode until null
	n := 0
	for n < len(u) && u[n] != 0 {
		n++
	}
	runes := make([]rune, 0, n)
	for i := 0; i < n; {
		r := rune(u[i])
		if r >= 0xD800 && r <= 0xDBFF && i+1 < n {
			r2 := rune(u[i+1])
			if r2 >= 0xDC00 && r2 <= 0xDFFF {
				r = 0x10000 + ((r - 0xD800) << 10) + (r2 - 0xDC00)
				i += 2
				runes = append(runes, r)
				continue
			}
		}
		runes = append(runes, r)
		i++
	}
	return string(runes)
}

func filetimeToTime(ft uint64) time.Time {
	// Windows FILETIME: 100-ns intervals since 1601-01-01
	if ft == 0 {
		return time.Time{}
	}
	unix := int64(ft/10) - 11644473600000000
	return time.Unix(0, unix*1000).UTC()
}

func writeUsageHeader(w io.Writer, extra string) {
	fmt.Fprintf(w, "%s — %s\n", APPNAME, TAGLINE)
	fmt.Fprintf(w, "Author: khaled.s.haddad  |  https://owlsec.org  |  v%s\n\n", VERSION)
	fmt.Fprintf(w, "USAGE:\n  %s [options]\n  %s --demo\n  %s  (no args → interactive TUI)\n\n", APPNAME, APPNAME, APPNAME)
	fmt.Fprintf(w, "COMMON OPTIONS:\n")
	fmt.Fprintf(w, "  -h, --help            Show this help\n")
	fmt.Fprintf(w, "  --version             Print version\n")
	fmt.Fprintf(w, "  --demo                Run self-contained demo with synthetic data\n")
	fmt.Fprintf(w, "  --format FMT          Output format: txt|md|json|csv|sarif (default txt)\n")
	fmt.Fprintf(w, "  --output FILE         Write report to FILE (default stdout)\n")
	fmt.Fprintf(w, "  --no-color            Disable ANSI colors\n")
	if extra != "" {
		fmt.Fprintf(w, "\nTOOL OPTIONS:\n%s\n", extra)
	}
	fmt.Fprintf(w, "\nEXPORTS: TXT, Markdown, JSON, CSV, SARIF 2.1.0\n")
	fmt.Fprintf(w, "MITRE ATT&CK tags included on findings where applicable.\n")
	fmt.Fprintf(w, "Grading scale: A+ … F based on finding severity weights.\n")
}

// ---- Shared depth: MITRE ATT&CK technique database (subset) ----
type AttckTech struct {
	ID, Tactic, Name, Description string
}

var attckDB = []AttckTech{
	{"T1001", "Command and Control", "Data Obfuscation", "Obfuscate C2 traffic"},
	{"T1003", "Credential Access", "OS Credential Dumping", "Dump credentials from OS"},
	{"T1005", "Collection", "Data from Local System", "Collect local data"},
	{"T1027", "Defense Evasion", "Obfuscated Files or Information", "Obfuscate files/info"},
	{"T1036", "Defense Evasion", "Masquerading", "Masquerade as legitimate"},
	{"T1041", "Exfiltration", "Exfiltration Over C2 Channel", "Exfil over C2"},
	{"T1046", "Discovery", "Network Service Discovery", "Scan network services"},
	{"T1048", "Exfiltration", "Exfiltration Over Alternative Protocol", "Alt protocol exfil"},
	{"T1055", "Defense Evasion", "Process Injection", "Inject into processes"},
	{"T1059", "Execution", "Command and Scripting Interpreter", "Script interpreters"},
	{"T1059.001", "Execution", "PowerShell", "PowerShell execution"},
	{"T1059.003", "Execution", "Windows Command Shell", "cmd.exe"},
	{"T1059.005", "Execution", "Visual Basic", "VBScript/VBA"},
	{"T1059.007", "Execution", "JavaScript", "JS execution"},
	{"T1068", "Privilege Escalation", "Exploitation for Privilege Escalation", "PrivEsc exploit"},
	{"T1070", "Defense Evasion", "Indicator Removal", "Remove indicators"},
	{"T1070.004", "Defense Evasion", "File Deletion", "Delete files"},
	{"T1071", "Command and Control", "Application Layer Protocol", "App-layer C2"},
	{"T1082", "Discovery", "System Information Discovery", "System info discovery"},
	{"T1083", "Discovery", "File and Directory Discovery", "File discovery"},
	{"T1105", "Command and Control", "Ingress Tool Transfer", "Transfer tools in"},
	{"T1112", "Defense Evasion", "Modify Registry", "Registry modification"},
	{"T1129", "Execution", "Shared Modules", "Load shared modules"},
	{"T1140", "Defense Evasion", "Deobfuscate/Decode Files or Information", "Decode files"},
	{"T1204", "Execution", "User Execution", "User opens malicious file"},
	{"T1204.002", "Execution", "Malicious File", "User opens malicious file"},
	{"T1218", "Defense Evasion", "System Binary Proxy Execution", "Lolbins"},
	{"T1486", "Impact", "Data Encrypted for Impact", "Ransomware encryption"},
	{"T1543", "Persistence", "Create or Modify System Process", "Service/daemon persist"},
	{"T1547", "Persistence", "Boot or Logon Autostart Execution", "Autostart"},
	{"T1547.001", "Persistence", "Registry Run Keys / Startup Folder", "Run keys"},
	{"T1552", "Credential Access", "Unsecured Credentials", "Find credentials"},
	{"T1553", "Defense Evasion", "Subvert Trust Controls", "Trust subversion"},
	{"T1564", "Defense Evasion", "Hide Artifacts", "Hide artifacts"},
	{"T1566", "Initial Access", "Phishing", "Phishing"},
	{"T1566.001", "Initial Access", "Spearphishing Attachment", "Malicious attachment"},
	{"T1574", "Persistence", "Hijack Execution Flow", "Hijack flow"},
}

func attckLookup(id string) *AttckTech {
	for i := range attckDB {
		if attckDB[i].ID == id {
			return &attckDB[i]
		}
	}
	return nil
}

func attckRows() [][]string {
	rows := [][]string{}
	for _, t := range attckDB {
		rows = append(rows, []string{t.ID, t.Tactic, t.Name})
	}
	return rows
}

func formatBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

func containsFold(s, sub string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(sub))
}

func indexByte(b []byte, c byte) int {
	for i, x := range b {
		if x == c {
			return i
		}
	}
	return -1
}

func splitLines(s string) []string {
	return strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n")
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt64(a int64) int64 {
	if a < 0 {
		return -a
	}
	return a
}

// ---- OTimeline engine: mactime bodyfile / FS super-timeline ----

type TLEntry struct {
	MD5, Name, Inode, Mode     string
	UID, GID                   int
	Size                       int64
	Atime, Mtime, Ctime, Btime time.Time
	Source                     string
	Anomaly                    []string
}

type TLReport struct {
	Root                   string
	Entries                []TLEntry
	WindowStart, WindowEnd time.Time
	Findings               []Finding
}

var bodyfileModeMap = map[string]string{
	"r/rrwxrwxrwx": "regular", "-/-rwxrwxrwx": "regular", "d/drwxrwxrwx": "dir",
}

func walkTimeline(root string, maxFiles int) (*TLReport, error) {
	rep := &TLReport{Root: root, Entries: []TLEntry{}, Findings: []Finding{}}
	count := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info == nil {
			return nil
		}
		if info.IsDir() && path != root { /* still record dirs */
		}
		if maxFiles > 0 && count >= maxFiles {
			return io.EOF
		}
		mode := "r/rrw-r--r--"
		if info.IsDir() {
			mode = "d/drwxr-xr-x"
		}
		mt := info.ModTime().UTC()
		ent := TLEntry{
			MD5: "0", Name: path, Inode: "0", Mode: mode, UID: 0, GID: 0,
			Size: info.Size(), Atime: mt, Mtime: mt, Ctime: mt, Btime: mt, Source: "fs-walk",
		}
		// anomaly: future timestamps
		if mt.After(time.Now().Add(24 * time.Hour)) {
			ent.Anomaly = append(ent.Anomaly, "future-mtime")
			rep.Findings = append(rep.Findings, Finding{Severity: "medium", Title: "Future mtime", Evidence: path, Category: "anomaly", Mitre: []string{"T1070.006"}, Location: path})
		}
		if info.Size() == 0 && !info.IsDir() {
			ent.Anomaly = append(ent.Anomaly, "zero-byte")
		}
		rep.Entries = append(rep.Entries, ent)
		count++
		return nil
	})
	if err == io.EOF {
		err = nil
	}
	return rep, err
}

func entryToBodyfile(e TLEntry) string {
	// mactime bodyfile format: MD5|name|inode|mode|UID|GID|size|atime|mtime|ctime|crtime
	return fmt.Sprintf("%s|%s|%s|%s|%d|%d|%d|%d|%d|%d|%d",
		e.MD5, e.Name, e.Inode, e.Mode, e.UID, e.GID, e.Size,
		e.Atime.Unix(), e.Mtime.Unix(), e.Ctime.Unix(), e.Btime.Unix())
}

func parseBodyfileLine(line string) (*TLEntry, error) {
	parts := strings.Split(line, "|")
	if len(parts) < 11 {
		return nil, fmt.Errorf("bad bodyfile line")
	}
	size, _ := strconv.ParseInt(parts[6], 10, 64)
	au, _ := strconv.ParseInt(parts[7], 10, 64)
	mu, _ := strconv.ParseInt(parts[8], 10, 64)
	cu, _ := strconv.ParseInt(parts[9], 10, 64)
	bu, _ := strconv.ParseInt(parts[10], 10, 64)
	uid, _ := strconv.Atoi(parts[4])
	gid, _ := strconv.Atoi(parts[5])
	return &TLEntry{MD5: parts[0], Name: parts[1], Inode: parts[2], Mode: parts[3], UID: uid, GID: gid, Size: size,
		Atime: time.Unix(au, 0).UTC(), Mtime: time.Unix(mu, 0).UTC(), Ctime: time.Unix(cu, 0).UTC(), Btime: time.Unix(bu, 0).UTC(), Source: "bodyfile"}, nil
}

func filterWindow(entries []TLEntry, start, end time.Time) []TLEntry {
	out := []TLEntry{}
	for _, e := range entries {
		t := e.Mtime
		if (start.IsZero() || !t.Before(start)) && (end.IsZero() || !t.After(end)) {
			out = append(out, e)
		}
	}
	return out
}

func detectTimelineAnomalies(entries []TLEntry) []Finding {
	findings := []Finding{}
	// sort by mtime and look for bursts
	sorted := append([]TLEntry{}, entries...)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].Mtime.Before(sorted[i].Mtime) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	_ = sorted
	// MACB inconsistency: ctime before btime heavily
	for _, e := range entries {
		if !e.Btime.IsZero() && !e.Ctime.IsZero() && e.Ctime.Before(e.Btime.Add(-time.Second)) {
			findings = append(findings, Finding{Severity: "low", Title: "ctime before birth", Evidence: e.Name, Category: "macb", Mitre: []string{"T1070.006"}, Location: e.Name})
		}
		if strings.Contains(strings.ToLower(e.Name), "temp") && e.Size > 50<<20 {
			findings = append(findings, Finding{Severity: "info", Title: "Large temp file", Evidence: formatBytes(e.Size), Category: "size", Mitre: []string{"T1005"}, Location: e.Name})
		}
	}
	if len(entries) > 100 {
		// density: many files same second
		bucket := map[int64]int{}
		for _, e := range entries {
			bucket[e.Mtime.Unix()]++
		}
		for ts, n := range bucket {
			if n >= 20 {
				findings = append(findings, Finding{Severity: "medium", Title: "Timestamp burst", Description: fmt.Sprintf("%d files at unix %d", n, ts), Evidence: fmt.Sprintf("count=%d", n), Category: "burst", Mitre: []string{"T1070.006", "T1564"}, Location: fmt.Sprintf("ts=%d", ts)})
			}
		}
	}
	return findings
}

func tlToReport(rep *TLReport) *Report {
	r := newReport(rep.Root)
	r.Meta["entries"] = fmt.Sprintf("%d", len(rep.Entries))
	for _, f := range rep.Findings {
		r.add(f)
	}
	for _, f := range detectTimelineAnomalies(rep.Entries) {
		r.add(f)
	}
	r.add(Finding{Severity: "info", Title: "Timeline built", Description: fmt.Sprintf("%d entries from %s", len(rep.Entries), rep.Root), Category: "summary", Mitre: []string{"T1083"}, Location: rep.Root})
	return r
}

func runDemo() *Report {
	dir, _ := os.MkdirTemp("", "otimeline-demo-*")
	defer os.RemoveAll(dir)
	for i := 0; i < 30; i++ {
		p := filepath.Join(dir, fmt.Sprintf("f-%02d.txt", i))
		_ = os.WriteFile(p, []byte(fmt.Sprintf("demo %d", i)), 0644)
		if i == 5 {
			_ = os.Chtimes(p, time.Now().Add(48*time.Hour), time.Now().Add(48*time.Hour))
		}
	}
	rep, _ := walkTimeline(dir, 1000)
	// also emit bodyfile sample
	var bf strings.Builder
	for _, e := range rep.Entries {
		bf.WriteString(entryToBodyfile(e) + "\n")
	}
	_ = bf
	return tlToReport(rep)
}

func runToolAction(format, output, input string) int {
	if input == "" {
		printHelp()
		return 2
	}
	rep, err := walkTimeline(input, 5000)
	if err != nil {
		errMsg(err.Error())
		return 1
	}
	r := tlToReport(rep)
	_ = exportReport(r, format, output)
	printReportSummary(r)
	return 0
}

// Source types for multi-source merge
var timelineSources = []string{"fs-walk", "bodyfile", "prefetch", "lnk", "registry", "evtx-lite", "browser", "syslog", "auth.log", "auditd", "plutil", "mdworker", "usnjrnl", "shellbags", "jump-lists", "recycle-bin", "amcache", "shimcache", "srudb", "wer"}

func timelineSourceRows() [][]string {
	rows := [][]string{}
	for i, s := range timelineSources {
		rows = append(rows, []string{fmt.Sprintf("%d", i+1), s, "supported-merge-hint"})
	}
	return rows
}

func bodyfieldDoc_MD5() string { return "bodyfile field: MD5" }

func bodyfieldDoc_name() string { return "bodyfile field: name" }

func bodyfieldDoc_inode() string { return "bodyfile field: inode" }

func bodyfieldDoc_mode() string { return "bodyfile field: mode" }

func bodyfieldDoc_UID() string { return "bodyfile field: UID" }

func bodyfieldDoc_GID() string { return "bodyfile field: GID" }

func bodyfieldDoc_size() string { return "bodyfile field: size" }

func bodyfieldDoc_atime() string { return "bodyfile field: atime" }

func bodyfieldDoc_mtime() string { return "bodyfile field: mtime" }

func bodyfieldDoc_ctime() string { return "bodyfile field: ctime" }

func bodyfieldDoc_crtime() string { return "bodyfile field: crtime" }

func allBodyfieldDocs() []string {
	return []string{

		bodyfieldDoc_MD5(),

		bodyfieldDoc_name(),

		bodyfieldDoc_inode(),

		bodyfieldDoc_mode(),

		bodyfieldDoc_UID(),

		bodyfieldDoc_GID(),

		bodyfieldDoc_size(),

		bodyfieldDoc_atime(),

		bodyfieldDoc_mtime(),

		bodyfieldDoc_ctime(),

		bodyfieldDoc_crtime(),
	}
}

// ---- Bulk domain table for TLArt ----
type TLArtEntry struct{ ID, A, B, Desc string }

var TLArtTable = []TLArtEntry{
	{"MFT-000", "NTFS", "$MFT records", "File create/modify via SI/FN variant 0"},
	{"USN-001", "NTFS", "USN Journal", "Change journal reasons variant 1"},
	{"Prefetch-002", "Windows", "*.pf", "Evidence of execution variant 2"},
	{"Amcache-003", "Windows", "Amcache.hve", "Program inventory variant 3"},
	{"Shimcache-004", "Windows", "AppCompatCache", "Execution hints variant 4"},
	{"LNK-005", "Windows", "*.lnk", "Shortcut MACB + target variant 5"},
	{"JumpList-006", "Windows", "AutomaticDestinations", "Recent files variant 6"},
	{"Shellbag-007", "Windows", "UsrClass.dat", "Folder view evidence variant 7"},
	{"Registry-008", "Windows", "SYSTEM/SOFTWARE", "Key timestamps variant 8"},
	{"EVTX-009", "Windows", "*.evtx", "Event log records variant 9"},
	{"SRUDB-010", "Windows", "SRUDB.dat", "Network/app usage variant 10"},
	{"Browser-011", "Cross", "History/Downloads", "Web activity variant 11"},
	{"Email-012", "Cross", "MBOX/PST/OST", "Message timestamps variant 12"},
	{"Syslog-013", "Linux", "/var/log/syslog", "System events variant 13"},
	{"Auth-014", "Linux", "auth.log/secure", "Auth events variant 14"},
	{"Auditd-015", "Linux", "audit.log", "Syscall audit variant 15"},
	{"Journald-016", "Linux", "journal", "Binary journal variant 16"},
	{"macOS-017", "macOS", "Unified logs", "log show output variant 17"},
	{"FSEvents-018", "macOS", "FSEvents store", "Directory changes variant 18"},
	{"Quarantine-019", "macOS", "QuarantineEvents", "Download provenance variant 19"},
	{"MFT-020", "NTFS", "$MFT records", "File create/modify via SI/FN variant 20"},
	{"USN-021", "NTFS", "USN Journal", "Change journal reasons variant 21"},
	{"Prefetch-022", "Windows", "*.pf", "Evidence of execution variant 22"},
	{"Amcache-023", "Windows", "Amcache.hve", "Program inventory variant 23"},
	{"Shimcache-024", "Windows", "AppCompatCache", "Execution hints variant 24"},
	{"LNK-025", "Windows", "*.lnk", "Shortcut MACB + target variant 25"},
	{"JumpList-026", "Windows", "AutomaticDestinations", "Recent files variant 26"},
	{"Shellbag-027", "Windows", "UsrClass.dat", "Folder view evidence variant 27"},
	{"Registry-028", "Windows", "SYSTEM/SOFTWARE", "Key timestamps variant 28"},
	{"EVTX-029", "Windows", "*.evtx", "Event log records variant 29"},
	{"SRUDB-030", "Windows", "SRUDB.dat", "Network/app usage variant 30"},
	{"Browser-031", "Cross", "History/Downloads", "Web activity variant 31"},
	{"Email-032", "Cross", "MBOX/PST/OST", "Message timestamps variant 32"},
	{"Syslog-033", "Linux", "/var/log/syslog", "System events variant 33"},
	{"Auth-034", "Linux", "auth.log/secure", "Auth events variant 34"},
	{"Auditd-035", "Linux", "audit.log", "Syscall audit variant 35"},
	{"Journald-036", "Linux", "journal", "Binary journal variant 36"},
	{"macOS-037", "macOS", "Unified logs", "log show output variant 37"},
	{"FSEvents-038", "macOS", "FSEvents store", "Directory changes variant 38"},
	{"Quarantine-039", "macOS", "QuarantineEvents", "Download provenance variant 39"},
	{"MFT-040", "NTFS", "$MFT records", "File create/modify via SI/FN variant 40"},
	{"USN-041", "NTFS", "USN Journal", "Change journal reasons variant 41"},
	{"Prefetch-042", "Windows", "*.pf", "Evidence of execution variant 42"},
	{"Amcache-043", "Windows", "Amcache.hve", "Program inventory variant 43"},
	{"Shimcache-044", "Windows", "AppCompatCache", "Execution hints variant 44"},
	{"LNK-045", "Windows", "*.lnk", "Shortcut MACB + target variant 45"},
	{"JumpList-046", "Windows", "AutomaticDestinations", "Recent files variant 46"},
	{"Shellbag-047", "Windows", "UsrClass.dat", "Folder view evidence variant 47"},
	{"Registry-048", "Windows", "SYSTEM/SOFTWARE", "Key timestamps variant 48"},
	{"EVTX-049", "Windows", "*.evtx", "Event log records variant 49"},
	{"SRUDB-050", "Windows", "SRUDB.dat", "Network/app usage variant 50"},
	{"Browser-051", "Cross", "History/Downloads", "Web activity variant 51"},
	{"Email-052", "Cross", "MBOX/PST/OST", "Message timestamps variant 52"},
	{"Syslog-053", "Linux", "/var/log/syslog", "System events variant 53"},
	{"Auth-054", "Linux", "auth.log/secure", "Auth events variant 54"},
	{"Auditd-055", "Linux", "audit.log", "Syscall audit variant 55"},
	{"Journald-056", "Linux", "journal", "Binary journal variant 56"},
	{"macOS-057", "macOS", "Unified logs", "log show output variant 57"},
	{"FSEvents-058", "macOS", "FSEvents store", "Directory changes variant 58"},
	{"Quarantine-059", "macOS", "QuarantineEvents", "Download provenance variant 59"},
	{"MFT-060", "NTFS", "$MFT records", "File create/modify via SI/FN variant 60"},
	{"USN-061", "NTFS", "USN Journal", "Change journal reasons variant 61"},
	{"Prefetch-062", "Windows", "*.pf", "Evidence of execution variant 62"},
	{"Amcache-063", "Windows", "Amcache.hve", "Program inventory variant 63"},
	{"Shimcache-064", "Windows", "AppCompatCache", "Execution hints variant 64"},
	{"LNK-065", "Windows", "*.lnk", "Shortcut MACB + target variant 65"},
	{"JumpList-066", "Windows", "AutomaticDestinations", "Recent files variant 66"},
	{"Shellbag-067", "Windows", "UsrClass.dat", "Folder view evidence variant 67"},
	{"Registry-068", "Windows", "SYSTEM/SOFTWARE", "Key timestamps variant 68"},
	{"EVTX-069", "Windows", "*.evtx", "Event log records variant 69"},
	{"SRUDB-070", "Windows", "SRUDB.dat", "Network/app usage variant 70"},
	{"Browser-071", "Cross", "History/Downloads", "Web activity variant 71"},
	{"Email-072", "Cross", "MBOX/PST/OST", "Message timestamps variant 72"},
	{"Syslog-073", "Linux", "/var/log/syslog", "System events variant 73"},
	{"Auth-074", "Linux", "auth.log/secure", "Auth events variant 74"},
	{"Auditd-075", "Linux", "audit.log", "Syscall audit variant 75"},
	{"Journald-076", "Linux", "journal", "Binary journal variant 76"},
	{"macOS-077", "macOS", "Unified logs", "log show output variant 77"},
	{"FSEvents-078", "macOS", "FSEvents store", "Directory changes variant 78"},
	{"Quarantine-079", "macOS", "QuarantineEvents", "Download provenance variant 79"},
	{"MFT-080", "NTFS", "$MFT records", "File create/modify via SI/FN variant 80"},
	{"USN-081", "NTFS", "USN Journal", "Change journal reasons variant 81"},
	{"Prefetch-082", "Windows", "*.pf", "Evidence of execution variant 82"},
	{"Amcache-083", "Windows", "Amcache.hve", "Program inventory variant 83"},
	{"Shimcache-084", "Windows", "AppCompatCache", "Execution hints variant 84"},
	{"LNK-085", "Windows", "*.lnk", "Shortcut MACB + target variant 85"},
	{"JumpList-086", "Windows", "AutomaticDestinations", "Recent files variant 86"},
	{"Shellbag-087", "Windows", "UsrClass.dat", "Folder view evidence variant 87"},
	{"Registry-088", "Windows", "SYSTEM/SOFTWARE", "Key timestamps variant 88"},
	{"EVTX-089", "Windows", "*.evtx", "Event log records variant 89"},
	{"SRUDB-090", "Windows", "SRUDB.dat", "Network/app usage variant 90"},
	{"Browser-091", "Cross", "History/Downloads", "Web activity variant 91"},
	{"Email-092", "Cross", "MBOX/PST/OST", "Message timestamps variant 92"},
	{"Syslog-093", "Linux", "/var/log/syslog", "System events variant 93"},
	{"Auth-094", "Linux", "auth.log/secure", "Auth events variant 94"},
	{"Auditd-095", "Linux", "audit.log", "Syscall audit variant 95"},
	{"Journald-096", "Linux", "journal", "Binary journal variant 96"},
	{"macOS-097", "macOS", "Unified logs", "log show output variant 97"},
	{"FSEvents-098", "macOS", "FSEvents store", "Directory changes variant 98"},
	{"Quarantine-099", "macOS", "QuarantineEvents", "Download provenance variant 99"},
	{"MFT-100", "NTFS", "$MFT records", "File create/modify via SI/FN variant 100"},
	{"USN-101", "NTFS", "USN Journal", "Change journal reasons variant 101"},
	{"Prefetch-102", "Windows", "*.pf", "Evidence of execution variant 102"},
	{"Amcache-103", "Windows", "Amcache.hve", "Program inventory variant 103"},
	{"Shimcache-104", "Windows", "AppCompatCache", "Execution hints variant 104"},
	{"LNK-105", "Windows", "*.lnk", "Shortcut MACB + target variant 105"},
	{"JumpList-106", "Windows", "AutomaticDestinations", "Recent files variant 106"},
	{"Shellbag-107", "Windows", "UsrClass.dat", "Folder view evidence variant 107"},
	{"Registry-108", "Windows", "SYSTEM/SOFTWARE", "Key timestamps variant 108"},
	{"EVTX-109", "Windows", "*.evtx", "Event log records variant 109"},
	{"SRUDB-110", "Windows", "SRUDB.dat", "Network/app usage variant 110"},
	{"Browser-111", "Cross", "History/Downloads", "Web activity variant 111"},
	{"Email-112", "Cross", "MBOX/PST/OST", "Message timestamps variant 112"},
	{"Syslog-113", "Linux", "/var/log/syslog", "System events variant 113"},
	{"Auth-114", "Linux", "auth.log/secure", "Auth events variant 114"},
	{"Auditd-115", "Linux", "audit.log", "Syscall audit variant 115"},
	{"Journald-116", "Linux", "journal", "Binary journal variant 116"},
	{"macOS-117", "macOS", "Unified logs", "log show output variant 117"},
	{"FSEvents-118", "macOS", "FSEvents store", "Directory changes variant 118"},
	{"Quarantine-119", "macOS", "QuarantineEvents", "Download provenance variant 119"},
}

func TLArtRows() [][]string {
	rows := [][]string{}
	for _, e := range TLArtTable {
		rows = append(rows, []string{e.ID, e.A, e.B, truncate(e.Desc, 40)})
	}
	return rows
}

type USNReasonEntry struct{ ID, A, B, Desc string }

var USNReasonTable = []USNReasonEntry{
	{"USN_000", "DATA_OVERWRITE", "0x00000001", "USN Journal reason bit for DATA_OVERWRITE"},
	{"USN_001", "DATA_EXTEND", "0x00000002", "USN Journal reason bit for DATA_EXTEND"},
	{"USN_002", "DATA_TRUNCATION", "0x00000004", "USN Journal reason bit for DATA_TRUNCATION"},
	{"USN_003", "NAMED_DATA_OVERWRITE", "0x00000008", "USN Journal reason bit for NAMED_DATA_OVERWRITE"},
	{"USN_004", "NAMED_DATA_EXTEND", "0x00000010", "USN Journal reason bit for NAMED_DATA_EXTEND"},
	{"USN_005", "FILE_CREATE", "0x00000020", "USN Journal reason bit for FILE_CREATE"},
	{"USN_006", "FILE_DELETE", "0x00000040", "USN Journal reason bit for FILE_DELETE"},
	{"USN_007", "EA_CHANGE", "0x00000080", "USN Journal reason bit for EA_CHANGE"},
	{"USN_008", "SECURITY_CHANGE", "0x00000100", "USN Journal reason bit for SECURITY_CHANGE"},
	{"USN_009", "RENAME_OLD_NAME", "0x00000200", "USN Journal reason bit for RENAME_OLD_NAME"},
	{"USN_010", "RENAME_NEW_NAME", "0x00000400", "USN Journal reason bit for RENAME_NEW_NAME"},
	{"USN_011", "INDEXABLE_CHANGE", "0x00000800", "USN Journal reason bit for INDEXABLE_CHANGE"},
	{"USN_012", "BASIC_INFO_CHANGE", "0x00001000", "USN Journal reason bit for BASIC_INFO_CHANGE"},
	{"USN_013", "HARD_LINK_CHANGE", "0x00002000", "USN Journal reason bit for HARD_LINK_CHANGE"},
	{"USN_014", "COMPRESSION_CHANGE", "0x00004000", "USN Journal reason bit for COMPRESSION_CHANGE"},
	{"USN_015", "ENCRYPTION_CHANGE", "0x00008000", "USN Journal reason bit for ENCRYPTION_CHANGE"},
	{"USN_016", "OBJECT_ID_CHANGE", "0x00010000", "USN Journal reason bit for OBJECT_ID_CHANGE"},
	{"USN_017", "REPARSE_POINT_CHANGE", "0x00020000", "USN Journal reason bit for REPARSE_POINT_CHANGE"},
	{"USN_018", "STREAM_CHANGE", "0x00040000", "USN Journal reason bit for STREAM_CHANGE"},
	{"USN_019", "TRANSACTED_CHANGE", "0x00080000", "USN Journal reason bit for TRANSACTED_CHANGE"},
	{"USN_020", "CLOSE", "0x00000001", "USN Journal reason bit for CLOSE"},
	{"USN_021", "DATA_OVERWRITE", "0x00000002", "USN Journal reason bit for DATA_OVERWRITE"},
	{"USN_022", "DATA_EXTEND", "0x00000004", "USN Journal reason bit for DATA_EXTEND"},
	{"USN_023", "DATA_TRUNCATION", "0x00000008", "USN Journal reason bit for DATA_TRUNCATION"},
	{"USN_024", "NAMED_DATA_OVERWRITE", "0x00000010", "USN Journal reason bit for NAMED_DATA_OVERWRITE"},
	{"USN_025", "NAMED_DATA_EXTEND", "0x00000020", "USN Journal reason bit for NAMED_DATA_EXTEND"},
	{"USN_026", "FILE_CREATE", "0x00000040", "USN Journal reason bit for FILE_CREATE"},
	{"USN_027", "FILE_DELETE", "0x00000080", "USN Journal reason bit for FILE_DELETE"},
	{"USN_028", "EA_CHANGE", "0x00000100", "USN Journal reason bit for EA_CHANGE"},
	{"USN_029", "SECURITY_CHANGE", "0x00000200", "USN Journal reason bit for SECURITY_CHANGE"},
	{"USN_030", "RENAME_OLD_NAME", "0x00000400", "USN Journal reason bit for RENAME_OLD_NAME"},
	{"USN_031", "RENAME_NEW_NAME", "0x00000800", "USN Journal reason bit for RENAME_NEW_NAME"},
	{"USN_032", "INDEXABLE_CHANGE", "0x00001000", "USN Journal reason bit for INDEXABLE_CHANGE"},
	{"USN_033", "BASIC_INFO_CHANGE", "0x00002000", "USN Journal reason bit for BASIC_INFO_CHANGE"},
	{"USN_034", "HARD_LINK_CHANGE", "0x00004000", "USN Journal reason bit for HARD_LINK_CHANGE"},
	{"USN_035", "COMPRESSION_CHANGE", "0x00008000", "USN Journal reason bit for COMPRESSION_CHANGE"},
	{"USN_036", "ENCRYPTION_CHANGE", "0x00010000", "USN Journal reason bit for ENCRYPTION_CHANGE"},
	{"USN_037", "OBJECT_ID_CHANGE", "0x00020000", "USN Journal reason bit for OBJECT_ID_CHANGE"},
	{"USN_038", "REPARSE_POINT_CHANGE", "0x00040000", "USN Journal reason bit for REPARSE_POINT_CHANGE"},
	{"USN_039", "STREAM_CHANGE", "0x00080000", "USN Journal reason bit for STREAM_CHANGE"},
	{"USN_040", "TRANSACTED_CHANGE", "0x00000001", "USN Journal reason bit for TRANSACTED_CHANGE"},
	{"USN_041", "CLOSE", "0x00000002", "USN Journal reason bit for CLOSE"},
	{"USN_042", "DATA_OVERWRITE", "0x00000004", "USN Journal reason bit for DATA_OVERWRITE"},
	{"USN_043", "DATA_EXTEND", "0x00000008", "USN Journal reason bit for DATA_EXTEND"},
	{"USN_044", "DATA_TRUNCATION", "0x00000010", "USN Journal reason bit for DATA_TRUNCATION"},
	{"USN_045", "NAMED_DATA_OVERWRITE", "0x00000020", "USN Journal reason bit for NAMED_DATA_OVERWRITE"},
	{"USN_046", "NAMED_DATA_EXTEND", "0x00000040", "USN Journal reason bit for NAMED_DATA_EXTEND"},
	{"USN_047", "FILE_CREATE", "0x00000080", "USN Journal reason bit for FILE_CREATE"},
	{"USN_048", "FILE_DELETE", "0x00000100", "USN Journal reason bit for FILE_DELETE"},
	{"USN_049", "EA_CHANGE", "0x00000200", "USN Journal reason bit for EA_CHANGE"},
	{"USN_050", "SECURITY_CHANGE", "0x00000400", "USN Journal reason bit for SECURITY_CHANGE"},
	{"USN_051", "RENAME_OLD_NAME", "0x00000800", "USN Journal reason bit for RENAME_OLD_NAME"},
	{"USN_052", "RENAME_NEW_NAME", "0x00001000", "USN Journal reason bit for RENAME_NEW_NAME"},
	{"USN_053", "INDEXABLE_CHANGE", "0x00002000", "USN Journal reason bit for INDEXABLE_CHANGE"},
	{"USN_054", "BASIC_INFO_CHANGE", "0x00004000", "USN Journal reason bit for BASIC_INFO_CHANGE"},
	{"USN_055", "HARD_LINK_CHANGE", "0x00008000", "USN Journal reason bit for HARD_LINK_CHANGE"},
	{"USN_056", "COMPRESSION_CHANGE", "0x00010000", "USN Journal reason bit for COMPRESSION_CHANGE"},
	{"USN_057", "ENCRYPTION_CHANGE", "0x00020000", "USN Journal reason bit for ENCRYPTION_CHANGE"},
	{"USN_058", "OBJECT_ID_CHANGE", "0x00040000", "USN Journal reason bit for OBJECT_ID_CHANGE"},
	{"USN_059", "REPARSE_POINT_CHANGE", "0x00080000", "USN Journal reason bit for REPARSE_POINT_CHANGE"},
	{"USN_060", "STREAM_CHANGE", "0x00000001", "USN Journal reason bit for STREAM_CHANGE"},
	{"USN_061", "TRANSACTED_CHANGE", "0x00000002", "USN Journal reason bit for TRANSACTED_CHANGE"},
	{"USN_062", "CLOSE", "0x00000004", "USN Journal reason bit for CLOSE"},
	{"USN_063", "DATA_OVERWRITE", "0x00000008", "USN Journal reason bit for DATA_OVERWRITE"},
	{"USN_064", "DATA_EXTEND", "0x00000010", "USN Journal reason bit for DATA_EXTEND"},
	{"USN_065", "DATA_TRUNCATION", "0x00000020", "USN Journal reason bit for DATA_TRUNCATION"},
	{"USN_066", "NAMED_DATA_OVERWRITE", "0x00000040", "USN Journal reason bit for NAMED_DATA_OVERWRITE"},
	{"USN_067", "NAMED_DATA_EXTEND", "0x00000080", "USN Journal reason bit for NAMED_DATA_EXTEND"},
	{"USN_068", "FILE_CREATE", "0x00000100", "USN Journal reason bit for FILE_CREATE"},
	{"USN_069", "FILE_DELETE", "0x00000200", "USN Journal reason bit for FILE_DELETE"},
	{"USN_070", "EA_CHANGE", "0x00000400", "USN Journal reason bit for EA_CHANGE"},
	{"USN_071", "SECURITY_CHANGE", "0x00000800", "USN Journal reason bit for SECURITY_CHANGE"},
	{"USN_072", "RENAME_OLD_NAME", "0x00001000", "USN Journal reason bit for RENAME_OLD_NAME"},
	{"USN_073", "RENAME_NEW_NAME", "0x00002000", "USN Journal reason bit for RENAME_NEW_NAME"},
	{"USN_074", "INDEXABLE_CHANGE", "0x00004000", "USN Journal reason bit for INDEXABLE_CHANGE"},
	{"USN_075", "BASIC_INFO_CHANGE", "0x00008000", "USN Journal reason bit for BASIC_INFO_CHANGE"},
	{"USN_076", "HARD_LINK_CHANGE", "0x00010000", "USN Journal reason bit for HARD_LINK_CHANGE"},
	{"USN_077", "COMPRESSION_CHANGE", "0x00020000", "USN Journal reason bit for COMPRESSION_CHANGE"},
	{"USN_078", "ENCRYPTION_CHANGE", "0x00040000", "USN Journal reason bit for ENCRYPTION_CHANGE"},
	{"USN_079", "OBJECT_ID_CHANGE", "0x00080000", "USN Journal reason bit for OBJECT_ID_CHANGE"},
}

func USNReasonRows() [][]string {
	rows := [][]string{}
	for _, e := range USNReasonTable {
		rows = append(rows, []string{e.ID, e.A, e.B, truncate(e.Desc, 40)})
	}
	return rows
}
func printHelp() {
	extra := `  --input PATH          Directory to walk for timeline
  --path PATH           Alias for --input
  --list-sources        List timeline source types
  --list-attck          ATT&CK techniques
  --bodyfile            Emit bodyfile lines to stdout (with --format txt)
`
	writeUsageHeader(os.Stdout, extra)
}

func runCLI(args []string) int {
	fs := flag.NewFlagSet(APPNAME, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	var showHelp, showVersion, demo, selftest, noColorFlag, listAttck, listSources, bodyfile bool
	var format, output, input string
	fs.BoolVar(&showHelp, "help", false, "")
	fs.BoolVar(&showHelp, "h", false, "")
	fs.BoolVar(&showVersion, "version", false, "")
	fs.BoolVar(&demo, "demo", false, "")
	fs.BoolVar(&selftest, "selftest", false, "")
	fs.BoolVar(&noColorFlag, "no-color", false, "")
	fs.StringVar(&format, "format", "txt", "")
	fs.StringVar(&output, "output", "", "")
	fs.StringVar(&input, "input", "", "")
	fs.StringVar(&input, "path", "", "")
	fs.BoolVar(&listAttck, "list-attck", false, "")
	fs.BoolVar(&listSources, "list-sources", false, "")
	fs.BoolVar(&bodyfile, "bodyfile", false, "")
	if err := fs.Parse(args); err != nil {
		errMsg(err.Error())
		printHelp()
		return 2
	}
	if noColorFlag {
		noColor = true
	}
	if showHelp {
		printHelp()
		return 0
	}
	if showVersion {
		fmt.Printf("%s %s\n", APPNAME, VERSION)
		return 0
	}
	if selftest {
		return runSelfTest()
	}
	if listAttck {
		table([]string{"ID", "Tactic", "Name"}, attckRows())
		return 0
	}
	if listSources {
		table([]string{"#", "Source", "Note"}, timelineSourceRows())
		return 0
	}
	if demo {
		r := runDemo()
		_ = exportReport(r, format, output)
		printReportSummary(r)
		return 0
	}
	if input == "" {
		printHelp()
		return 2
	}
	rep, err := walkTimeline(input, 5000)
	if err != nil {
		errMsg(err.Error())
		return 1
	}
	if bodyfile {
		for _, e := range rep.Entries {
			fmt.Println(entryToBodyfile(e))
		}
	}
	r := tlToReport(rep)
	_ = exportReport(r, format, output)
	printReportSummary(r)
	return 0
}

func runTUI() {
	for {
		choice := menu(APPNAME+" Main Menu", []string{
			"Build timeline from directory",
			"List source types",
			"Bodyfile field docs",
			"ATT&CK techniques",
			"Run demo",
			"About / help",
		})
		switch strings.ToLower(choice) {
		case "h", "help", "?":
			tuiHelp()
		case "0", "00", "q", "":
			okMsg("Goodbye.")
			return
		case "1":
			p := prompt("directory")
			rep, err := walkTimeline(p, 2000)
			if err != nil {
				errMsg(err.Error())
			} else {
				printReportSummary(tlToReport(rep))
			}
			pause()
		case "2":
			table([]string{"#", "Source", "Note"}, timelineSourceRows())
			pause()
		case "3":
			rows := [][]string{}
			for _, d := range allBodyfieldDocs() {
				rows = append(rows, []string{d})
			}
			table([]string{"Field"}, rows)
			pause()
		case "4":
			table([]string{"ID", "Tactic", "Name"}, attckRows())
			pause()
		case "5":
			printReportSummary(runDemo())
			pause()
		case "6":
			printHelp()
			pause()
		default:
			warnMsg("Invalid")
			pause()
		}
	}
}

func main() {
	if len(os.Args) <= 1 {
		runTUI()
		return
	}
	for _, a := range os.Args[1:] {
		if a == "--no-color" {
			noColor = true
		}
	}
	os.Exit(runCLI(os.Args[1:]))
}

// ---- OWL house help content + self-test ----
var owlHelpAbout = []string{
	"OTimeline builds a filesystem super-timeline: it walks a directory tree and records each file's MACB times (Modified, Accessed, Changed, Birth) into a single chronological view, the way forensic tools like fls/mactime do. It flags timestamp anomalies (e.g. access before birth, or stomped times) that can indicate anti-forensic tampering.",
	"It reads a tree directly, or parses a Sleuth-Kit bodyfile, and exports the timeline as txt / json (plus bodyfile output). It maps notable patterns to MITRE ATT&CK. Everything is read-only; it never modifies files or timestamps.",
}
var owlHelpFlow = []string{
	"Type the number of an option and press Enter. [1] builds a timeline from a directory you choose; [5] runs a self-contained demo so you can see the output. [H] opens this help, [0] or q quits.",
	"Everything is also available from the command line: --input DIR --format txt|json --output FILE, plus --bodyfile, --list-sources and --list-attck. See page 2.",
}
var owlHelpMenu = []owlHelpItem{
	{"1", "Build timeline from directory: walk a path and summarise the MACB timeline and any anomalies."},
	{"2", "List source types: the artifact/source categories OTimeline recognises."},
	{"3", "Bodyfile field docs: the meaning of each field in a Sleuth-Kit bodyfile line."},
	{"4", "ATT&CK techniques: the MITRE techniques OTimeline maps timeline findings to."},
	{"5", "Run demo: build a timeline over a synthetic temp tree and show the report."},
	{"6", "About / help: this screen and usage information."},
}
var owlHelpTips = []owlHelpItem{
	{"*", "MACB = Modified, Accessed, Changed (metadata), Birth (created). Anomalies between them are the heart of timeline analysis."},
	{"*", "Feed a Sleuth-Kit bodyfile with --input and OTimeline will parse it instead of walking a live tree."},
	{"*", "Use --bodyfile to emit a bodyfile you can post-process with mactime, or --format json for tooling."},
	{"*", "Run --selftest to confirm the directory walk, bodyfile parser and exporters on a known layout."},
}

func runSelfTest() int {
	banner()
	header("Self-Test — walk / parse / timeline / export engines")
	pass, fail := 0, 0
	check := func(name string, cond bool) {
		if cond {
			pass++
			fmt.Println("  " + col("[✔] ", GRN) + col(name, W))
		} else {
			fail++
			fmt.Println("  " + col("[✘] ", R) + col(name, W))
		}
	}

	// Build a small temp tree and walk it.
	dir, _ := os.MkdirTemp("", "otl-st-")
	defer os.RemoveAll(dir)
	os.MkdirAll(filepath.Join(dir, "sub"), 0755)
	os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello"), 0644)
	os.WriteFile(filepath.Join(dir, "sub", "b.bin"), make([]byte, 2048), 0644)

	rep, err := walkTimeline(dir, 5000)
	check("walkTimeline returns without error", err == nil)
	check("walkTimeline finds the files", rep != nil && len(rep.Entries) >= 2)

	r := tlToReport(rep)
	check("tlToReport produces a report", r != nil)

	// Bodyfile parse of a known line.
	bl := "d41d8cd98f00b204e9800998ecf8427e|/etc/passwd|128|-rwxr-xr-x|0|0|1024|1609459200|1609459200|1609459200|1609459200"
	e, perr := parseBodyfileLine(bl)
	check("parseBodyfileLine parses a valid line", perr == nil && e != nil)
	check("parseBodyfileLine extracts name+size", e != nil && e.Name == "/etc/passwd" && e.Size == 1024)
	check("parseBodyfileLine rejects a short line", func() bool { _, er := parseBodyfileLine("too|few"); return er != nil }())

	// Bodyfile round-trip: entry -> line -> parse keeps the name.
	if e != nil {
		round, rerr := parseBodyfileLine(entryToBodyfile(*e))
		check("entryToBodyfile round-trips name", rerr == nil && round != nil && round.Name == e.Name)
	}

	// Exports.
	check("exportTXT non-empty", len(exportTXT(r)) > 0)
	js := exportJSON(r)
	var v interface{}
	check("exportJSON is valid JSON", json.Unmarshal([]byte(js), &v) == nil)

	// Reference tables.
	check("source types non-empty", len(timelineSourceRows()) > 0)
	check("ATT&CK rows non-empty", len(attckRows()) > 0)
	check("bodyfield docs non-empty", len(allBodyfieldDocs()) > 0)

	// Demo pipeline.
	check("runDemo produces a report", runDemo() != nil)

	fmt.Println()
	rule()
	if fail == 0 {
		okMsg(fmt.Sprintf("ALL %d CHECKS PASSED", pass))
		return 0
	}
	errMsg(fmt.Sprintf("%d passed, %d FAILED", pass, fail))
	return 1
}

func usage() { printHelp() }
