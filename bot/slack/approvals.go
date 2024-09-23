package slack

import (
	"fmt"
	"github.com/keel-hq/keel/bot"
	log "github.com/sirupsen/logrus"
	"strings"
	"unicode"

	"github.com/keel-hq/keel/types"
	"github.com/slack-go/slack"
)

// Request - request approval
func (b *Bot) RequestApproval(req *types.Approval) error {
	return b.postApprovalMessageBlock(
		req.ID,
		createBlockMessage("Approval required! :mega:", b.name, req),
	)
}

func (b *Bot) ReplyToApproval(approval *types.Approval) error {
	var title string
	switch approval.Status() {
	case types.ApprovalStatusPending:
		title = "Approval required! :mega:"
	case types.ApprovalStatusRejected:
		title = "Change rejected! :negative_squared_cross_mark:"
	case types.ApprovalStatusApproved:
		title = "Change approved! :tada:"
	}

	b.upsertApprovalMessage(approval.ID, createBlockMessage(title, b.name, approval))
	return nil
}

func createBlockMessage(title string, botName string, req *types.Approval) slack.Blocks {
	if req.Expired() {
		title = title + " (Expired)"
	}

	headerText := slack.NewTextBlockObject(
		"plain_text",
		title,
		true,
		false,
	)
	headerSection := slack.NewHeaderBlock(headerText)

	messageSection := slack.NewTextBlockObject(
		"mrkdwn",
		req.Message,
		false,
		false,
	)
	messageBlock := slack.NewSectionBlock(messageSection, nil, nil)

	votesField := slack.NewTextBlockObject(
		"mrkdwn",
		fmt.Sprintf("*Votes:*\n%d/%d", req.VotesReceived, req.VotesRequired),
		false,
		false,
	)
	deltaField := slack.NewTextBlockObject(
		"mrkdwn",
		"*Delta:*\n"+req.Delta(),
		false,
		false,
	)
	leftDetailSection := slack.NewSectionBlock(
		nil,
		[]*slack.TextBlockObject{
			votesField,
			deltaField,
		},
		nil,
	)

	identifierField := slack.NewTextBlockObject(
		"mrkdwn",
		"*Identifier:*\n"+req.Identifier,
		false,
		false,
	)
	providerField := slack.NewTextBlockObject(
		"mrkdwn",
		"*Provider:*\n"+req.Provider.String(),
		false,
		false,
	)
	rightDetailSection := slack.NewSectionBlock(nil, []*slack.TextBlockObject{identifierField, providerField}, nil)

	commands := bot.BotEventTextToResponse["help"]
	var commandTexts []slack.MixedElement

	for i, cmd := range commands {
		// -- avoid adding first line in commands which is the title.
		if i == 0 {
			continue
		}
		cmd = addBotMentionToCommand(cmd, botName)
		commandTexts = append(commandTexts, slack.NewTextBlockObject("mrkdwn", cmd, false, false))
	}
	commandsBlock := slack.NewContextBlock("", commandTexts...)
	header := commands[0]

	blocks := []slack.Block{
		headerSection,
		messageBlock,
		leftDetailSection,
		rightDetailSection,
		slack.NewDividerBlock(),
		slack.NewContextBlock("", slack.NewTextBlockObject("mrkdwn", header, false, false)),
		commandsBlock,
	}

	if req.VotesReceived < req.VotesRequired && !req.Expired() && !req.Rejected {
		approveButton := slack.NewButtonBlockElement(
			bot.ApprovalResponseKeyword,
			req.Identifier,
			slack.NewTextBlockObject(
				"plain_text",
				"Approve",
				true,
				false,
			),
		)
		approveButton.Style = slack.StylePrimary

		rejectButton := slack.NewButtonBlockElement(
			bot.RejectResponseKeyword,
			req.Identifier,
			slack.NewTextBlockObject(
				"plain_text",
				"Reject",
				true,
				false,
			),
		)
		rejectButton.Style = slack.StyleDanger

		actionBlock := slack.NewActionBlock("", approveButton, rejectButton)

		blocks = append(
			blocks,
			slack.NewDividerBlock(),
			actionBlock,
		)
	}
	return slack.Blocks{
		BlockSet: blocks,
	}
}

func addBotMentionToCommand(command string, botName string) string {
	// -- retrieve the first letter of the command in order to insert bot mention
	firstLetterPos := -1
	for i, r := range command {
		if unicode.IsLetter(r) {
			firstLetterPos = i
			break
		}
	}

	if firstLetterPos < 0 {
		log.Debugf("Unable to find the first letter of the command '%s', let the command without the bot mention.", command)
		return command
	}

	return strings.Replace(
		command[:firstLetterPos]+fmt.Sprintf("@%s ", botName)+command[firstLetterPos:],
		"\"",
		"`",
		-1,
	)
}
