// Coletor de WhatsApp da Lúcia — SOMENTE LEITURA.
//
// Conecta como "aparelho conectado" (igual ao WhatsApp Web) e grava num banco local
// apenas as mensagens das conversas listadas em permitidas.txt. Não existe neste
// programa nenhuma chamada de envio de mensagem, arquivo, reação ou confirmação de leitura.
//
// Uso:
//
//	coletor rodar                 conecta (mostra QR na primeira vez) e fica gravando
//	coletor listar                mostra as conversas recentes (só nome e data, sem conteúdo)
//	coletor exportar -desde T     imprime em JSON as mensagens gravadas desde T (RFC3339)
package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/mdp/qrterminal/v3"
	qrcode "github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	_ "modernc.org/sqlite"
)

const (
	retencaoDias = 14
	maxTexto     = 800
)

var (
	pasta  string
	reLink = regexp.MustCompile(`(?i)\b(?:https?://|www\.)\S+`)
)

func main() {
	exe, _ := os.Executable()
	pasta = filepath.Dir(exe)
	if len(os.Args) < 2 {
		fmt.Println("uso: coletor rodar | listar | exportar -desde <RFC3339>")
		os.Exit(2)
	}
	switch os.Args[1] {
	case "rodar":
		rodar()
	case "listar":
		listar()
	case "exportar":
		fs := flag.NewFlagSet("exportar", flag.ExitOnError)
		desde := fs.String("desde", "", "data/hora RFC3339")
		_ = fs.Parse(os.Args[2:])
		exportar(*desde)
	default:
		fmt.Println("comando desconhecido:", os.Args[1])
		os.Exit(2)
	}
}

func abrirMensagens() *sql.DB {
	_ = os.MkdirAll(filepath.Join(pasta, "store"), 0o700)
	db, err := sql.Open("sqlite", "file:"+filepath.Join(pasta, "store", "mensagens.db")+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(`
CREATE TABLE IF NOT EXISTS conversas (jid TEXT PRIMARY KEY, nome TEXT, grupo INTEGER, ultima TEXT);
CREATE TABLE IF NOT EXISTS mensagens (id TEXT PRIMARY KEY, chat TEXT, chat_nome TEXT, remetente TEXT,
  de_mim INTEGER, quando TEXT, texto TEXT);
CREATE INDEX IF NOT EXISTS idx_msg_quando ON mensagens(quando);`)
	if err != nil {
		panic(err)
	}
	return db
}

// permitidas.txt: um número por linha (só dígitos, com DDI, ex. 5543999990000) ou JID; # comenta.
// Relido a cada mensagem: editar o arquivo vale na hora, sem reiniciar.
func permitidas() map[string]bool {
	m := map[string]bool{}
	b, err := os.ReadFile(filepath.Join(pasta, "permitidas.txt"))
	if err != nil {
		return m
	}
	for _, l := range strings.Split(string(b), "\n") {
		l = strings.TrimSpace(strings.SplitN(l, "#", 2)[0])
		if l != "" {
			m[strings.SplitN(l, "@", 2)[0]] = true
		}
	}
	return m
}

func nomeDe(ctx context.Context, cli *whatsmeow.Client, jid types.JID, push string) string {
	if c, err := cli.Store.Contacts.GetContact(ctx, jid); err == nil {
		for _, n := range []string{c.FullName, c.FirstName, c.BusinessName, c.PushName} {
			if n != "" {
				return n
			}
		}
	}
	if push != "" {
		return push
	}
	return jid.User
}

func textoDe(e *events.Message) string {
	m := e.Message
	switch {
	case m.GetConversation() != "":
		return m.GetConversation()
	case m.GetExtendedTextMessage().GetText() != "":
		return m.GetExtendedTextMessage().GetText()
	case m.GetImageMessage() != nil:
		return "[imagem] " + m.GetImageMessage().GetCaption()
	case m.GetVideoMessage() != nil:
		return "[vídeo] " + m.GetVideoMessage().GetCaption()
	case m.GetAudioMessage() != nil:
		return "[áudio]"
	case m.GetDocumentMessage() != nil:
		return "[documento] " + m.GetDocumentMessage().GetFileName()
	case m.GetLocationMessage() != nil:
		return "[localização]"
	case m.GetContactMessage() != nil:
		return "[contato]"
	}
	return ""
}

// Links viram "[link]" e o texto é cortado: nada clicável nem longo chega à Lúcia.
func limpar(t string) string {
	t = reLink.ReplaceAllString(t, "[link]")
	if r := []rune(t); len(r) > maxTexto {
		t = string(r[:maxTexto]) + "…"
	}
	return strings.TrimSpace(t)
}

func rodar() {
	ctx := context.Background()
	_ = os.MkdirAll(filepath.Join(pasta, "store"), 0o700)
	logWA := waLog.Stdout("WhatsApp", "WARN", true)
	container, err := sqlstore.New(ctx, "sqlite", "file:"+filepath.Join(pasta, "store", "whatsapp.db")+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", logWA)
	if err != nil {
		panic(err)
	}
	device, err := container.GetFirstDevice(ctx)
	if err != nil {
		panic(err)
	}
	cli := whatsmeow.NewClient(device, logWA)
	db := abrirMensagens()

	cli.AddEventHandler(func(evt any) {
		switch e := evt.(type) {
		case *events.Message:
			chat := e.Info.Chat
			nomeChat := nomeDe(ctx, cli, chat, "")
			if e.Info.IsGroup {
				var n string
				if db.QueryRow(`SELECT nome FROM conversas WHERE jid=?`, chat.String()).Scan(&n) == nil && n != "" {
					nomeChat = n
				}
			} else if !e.Info.IsFromMe {
				nomeChat = nomeDe(ctx, cli, chat, e.Info.PushName)
			}
			quando := e.Info.Timestamp.Format(time.RFC3339)
			// Metadados (sem conteúdo) de todas as conversas, para o Sandro escolher quais liberar.
			_, _ = db.Exec(`INSERT INTO conversas(jid,nome,grupo,ultima) VALUES(?,?,?,?)
				ON CONFLICT(jid) DO UPDATE SET nome=excluded.nome, ultima=excluded.ultima`,
				chat.String(), nomeChat, e.Info.IsGroup, quando)
			if !permitidas()[chat.User] {
				return
			}
			texto := limpar(textoDe(e))
			if texto == "" {
				return
			}
			remetente := "Sandro"
			if !e.Info.IsFromMe {
				remetente = nomeDe(ctx, cli, e.Info.Sender, e.Info.PushName)
			}
			_, _ = db.Exec(`INSERT OR IGNORE INTO mensagens(id,chat,chat_nome,remetente,de_mim,quando,texto)
				VALUES(?,?,?,?,?,?,?)`, e.Info.ID, chat.String(), nomeChat, remetente, e.Info.IsFromMe, quando, texto)
		case *events.Connected:
			// Fica "offline" para o celular continuar recebendo as notificações normalmente.
			_ = cli.SendPresence(ctx, types.PresenceUnavailable)
			fmt.Println(time.Now().Format("15:04"), "conectado ao WhatsApp (somente leitura)")
			// Só os nomes dos grupos (consulta de leitura), para o Sandro poder escolher pelo nome.
			go func() {
				grupos, err := cli.GetJoinedGroups(ctx)
				if err != nil {
					fmt.Println("não consegui listar os grupos:", err)
					return
				}
				for _, g := range grupos {
					_, _ = db.Exec(`INSERT INTO conversas(jid,nome,grupo,ultima) VALUES(?,?,1,'')
						ON CONFLICT(jid) DO UPDATE SET nome=excluded.nome`, g.JID.String(), g.Name)
				}
				fmt.Println(len(grupos), "grupos conhecidos")
			}()
		case *events.LoggedOut:
			fmt.Println("O WhatsApp desconectou este aparelho. Rode de novo e leia o QR code.")
			os.Exit(1)
		}
	})

	if cli.Store.ID == nil {
		qrCh, _ := cli.GetQRChannel(ctx)
		if err := cli.Connect(); err != nil {
			panic(err)
		}
		png := filepath.Join(pasta, "qr.png")
		for item := range qrCh {
			if item.Event == "code" {
				fmt.Println("\nNo celular: WhatsApp > Aparelhos conectados > Conectar um aparelho, e leia o QR:")
				qrterminal.GenerateHalfBlock(item.Code, qrterminal.L, os.Stdout)
				if qrcode.WriteFile(item.Code, qrcode.Medium, 512, png) == nil {
					_ = exec.Command("cmd", "/c", "start", "", png).Start()
				}
			} else {
				fmt.Println("pareamento:", item.Event)
			}
		}
		_ = os.Remove(png)
	} else if err := cli.Connect(); err != nil {
		panic(err)
	}

	// Retenção curta: apaga conteúdo com mais de 14 dias, agora e uma vez por dia.
	go func() {
		for {
			_, _ = db.Exec(`DELETE FROM mensagens WHERE quando < ?`, time.Now().AddDate(0, 0, -retencaoDias).Format(time.RFC3339))
			time.Sleep(24 * time.Hour)
		}
	}()

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig
	cli.Disconnect()
}

func listar() {
	db := abrirMensagens()
	perm := permitidas()
	rows, err := db.Query(`SELECT jid, nome, grupo, ultima FROM conversas ORDER BY ultima DESC LIMIT 300`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	fmt.Println("liberada | ultima msg  | tipo   | numero                | nome")
	for rows.Next() {
		var jid, nome, ultima string
		var grupo bool
		_ = rows.Scan(&jid, &nome, &grupo, &ultima)
		tipo, lib := "pessoa", "   "
		if grupo {
			tipo = "grupo "
		}
		if perm[strings.SplitN(jid, "@", 2)[0]] {
			lib = "SIM"
		}
		t, _ := time.Parse(time.RFC3339, ultima)
		fmt.Printf("   %s   | %s | %s | %-21s | %s\n", lib, t.Local().Format("02/01 15:04"), tipo, strings.SplitN(jid, "@", 2)[0], nome)
	}
}

func exportar(desde string) {
	t, err := time.Parse(time.RFC3339, desde)
	if err != nil {
		t = time.Now().Add(-3 * time.Hour)
	}
	db := abrirMensagens()
	rows, err := db.Query(`SELECT chat_nome, remetente, de_mim, quando, texto FROM mensagens WHERE quando >= ? ORDER BY quando`, t.Format(time.RFC3339))
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	type msg struct {
		Conversa  string `json:"conversa"`
		Remetente string `json:"remetente"`
		DeMim     bool   `json:"de_mim"`
		Quando    string `json:"quando"`
		Texto     string `json:"texto"`
	}
	out := []msg{}
	for rows.Next() {
		var m msg
		_ = rows.Scan(&m.Conversa, &m.Remetente, &m.DeMim, &m.Quando, &m.Texto)
		out = append(out, m)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	_ = enc.Encode(out)
}
