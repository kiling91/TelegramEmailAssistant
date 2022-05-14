package telegrammsg

import (
	"fmt"
	"github.com/kiling91/telegram-email-assistant/internal/common"
	"github.com/kiling91/telegram-email-assistant/internal/email"
	"github.com/kiling91/telegram-email-assistant/internal/factory"
	"github.com/kiling91/telegram-email-assistant/internal/print_msg"
	"strings"
)

type service struct {
	fact factory.Factory
}

func NewPrintEmail(fact factory.Factory) printmsg.PrintMsg {
	return &service{fact: fact}
}

func (s *service) PrintMsgEnvelope(msg *email.MessageEnvelope) string {
	result := ""
	if msg.ToName != "" {
		result += fmt.Sprintf("<b>📫 %s</b>\t (%s)\n\n", msg.ToName, msg.ToAddress)
	} else {
		result += fmt.Sprintf("<b>📫 %s</b>\n\n", msg.ToAddress)
	}

	if msg.FromName != "" {
		result += fmt.Sprintf("<b>📨 %s</b>\t (%s)\n\n", msg.FromName, msg.FromAddress)
	} else {
		result += fmt.Sprintf("<b>📨 %s</b>\n\n", msg.FromAddress)
	}

	result += fmt.Sprintf("⏰ <b>%s</b>\n\n", msg.Date.Format("2006-01-02 15:04"))
	result += fmt.Sprintf("📝 <b>%s</b>\n\n", msg.Subject)

	return result
}

func (s *service) needDrawHtml(msg *email.Message) bool {
	cfg := s.fact.Config()

	if msg.Body.TextPlain == "" {
		return true
	}

	if len(msg.Body.TextPlain) > cfg.MaxTextMessageSize {
		return true
	}

	if strings.Contains(msg.Body.TextHtml, "src=\"cid:") {
		return true
	}

	return false
}

func (s *service) PrintMsgWithBody(msg *email.Message) (*printmsg.FormattedMsg, error) {
	text := s.PrintMsgEnvelope(msg.Envelope)
	img := ""
	attachment := make([]string, 0)

	if s.needDrawHtml(msg) {
		cfg := s.fact.Config()
		dir, err := common.CreateFolderForEmail(cfg.FileStorageDir, msg.Envelope.ToAddress, msg.Uid)
		if err != nil {
			return nil, err
		}

		img, err = common.HtmlToPng(msg.Body.TextHtml, dir)
		if err != nil {
			return nil, fmt.Errorf("error convert html to png with error: %w", err)
		}
	} else {
		text += msg.Body.TextPlain
	}

	for _, att := range msg.Body.AttachmentFiles {
		text += fmt.Sprintf("\n📎 %s", att.FileName)
		attachment = append(attachment, att.FilePath)
	}

	return &printmsg.FormattedMsg{
		Text:       text,
		Img:        img,
		Attachment: attachment,
	}, nil
}
