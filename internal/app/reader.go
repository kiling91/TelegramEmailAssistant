package app

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/kiling91/telegram-email-assistant/internal/email"
	"github.com/kiling91/telegram-email-assistant/internal/factory"
	"github.com/kiling91/telegram-email-assistant/pkg/bot"
	"github.com/sirupsen/logrus"
)

type Reader struct {
	fact     factory.Factory
	userIds  []int64
	imapUser *email.ImapUser
}

func NewReader(fact factory.Factory, userIds []int64, imapUser *email.ImapUser) *Reader {
	return &Reader{
		fact:     fact,
		userIds:  userIds,
		imapUser: imapUser,
	}
}

func (r *Reader) startReadEmailBody(ctx context.Context, userId int64, msgUID int64) {
	b := r.fact.Bot()
	imap := r.fact.ImapEmail()
	pnt := r.fact.PrintMsg()
	login := r.fact.Config().Imap.Login
	// ⌛ Reading a mail from {fromEmail}
	// ⏳ Reading a mail from {fromEmail} ({seconds} sec)
	// ⌛ Reading a mail from {fromEmail} ({seconds} sec)
	edit, err := b.Send(userId, fmt.Sprintf("⌛ Reading a mail from id:%d", msgUID))
	defer b.Delete(edit)

	if err != nil {
		logrus.Warnf("error send msg to user %d", userId)
		return
	}

	msg, err := imap.ReadEmail(ctx, r.imapUser, msgUID)
	if err != nil {
		logrus.Warnf("error read msg #%d: %v", msgUID, err)
		return
	}

	fmsg, err := pnt.PrintMsgWithBody(msg, login)
	if err != nil {
		logrus.Warnf("error print msg #%d: %v", msgUID, err)
		return
	}

	if fmsg.Img != "" {
		_, err := b.SendPhoto(userId, &bot.Photo{
			Filename: fmsg.Img,
			Caption:  fmsg.Text,
		})
		if err != nil {
			logrus.Warnf("error send photo #%d: %v", msgUID, err)
			return
		}
	} else {
		_, err := b.Send(userId, fmsg.Text)
		if err != nil {
			logrus.Warnf("error send photo #%d: %v", msgUID, err)
			return
		}
	}

	for _, attach := range fmsg.Attachment {
		err := b.SendDocument(userId, attach)
		if err != nil {
			logrus.Warnf("error send document #%d: %v", msgUID, err)
			return
		}
	}
}

func (r *Reader) onButton(ctx context.Context, btnCtx bot.BtnContext) error {
	msgUID, err := strconv.ParseInt(btnCtx.Data(), 10, 32)
	if err != nil {
		logrus.Warnf("err parse string to int64: %v", err)
	}
	switch btnCtx.Unique() {
	case BtnMark:
	case BtnRead:
		go r.startReadEmailBody(ctx, btnCtx.UserId(), msgUID)
	default:
		logrus.Warnf("unknow btn type %s", btnCtx.Unique())
	}
	return nil
}

func (r *Reader) Start(ctx context.Context) {
	logrus.Info("Start read unseen emails")
	imap := r.fact.ImapEmail()
	b := r.fact.Bot()
	pnt := r.fact.PrintMsg()
	storage := r.fact.Storage()

	emails, err := imap.ReadUnseenEmails(ctx, r.imapUser)
	if err != nil {
		logrus.Fatalln(err)
	}

	sort.Slice(emails, func(i, j int) bool {
		return emails[i].Date.Before(emails[j].Date)
	})

	for _, e := range emails {
		if contains, err := storage.MsgIdContains(r.imapUser.Login, e.Uid); err != nil {
			logrus.Warnf("error get msg contains from storage: %v", err)
		} else if contains {
			continue
		}
		sid := strconv.FormatUint(uint64(e.Uid), 10)
		msg := pnt.PrintMsgEnvelope(e)
		for _, id := range r.userIds {
			inline := bot.NewInline(2, func(bc bot.BtnContext) error {
				return r.onButton(ctx, bc)
			})
			inline.Add("📩 Mark as read", BtnMark, sid)
			inline.Add("📧 Read", BtnRead, sid)
			if _, err := b.Send(id, msg, inline); err != nil {
				logrus.Warnf("error send msg: %v", err)
			} else {
				if err := storage.SaveMsgId(r.imapUser.Login, e.Uid); err != nil {
					logrus.Warnf("error save msg id to storage: %v", err)
				}
			}
		}
	}
}
