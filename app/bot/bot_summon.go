package bot

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/etcinit/gonduit/requests"
	"github.com/etcinit/phabulous/app/messages"
	"github.com/nlopes/slack"
)

// HandleSummon shows usage tip.
func (b *Bot) HandleSummon(ev *slack.MessageEvent, matches []string) {
	if len(matches) < 2 {
		b.Slacker.SimplePost(
			ev.Channel,
			"Usage: `summon <REVISION>`. Example: `summon D456`.",
			messages.IconDefault,
			true,
		)

		return
	}

	conn, err := b.Slacker.Factory.Make()
	if err != nil {
		b.Excuse(ev, err)
		return
	}

	id, err := strconv.Atoi(matches[1])
	if err != nil {
		b.Excuse(ev, err)
		return
	}

	res, err := conn.DifferentialQuery(requests.DifferentialQueryRequest{
		IDs: []uint64{uint64(id)},
	})
	if err != nil {
		b.Excuse(ev, err)
		return
	}

	if len(*res) == 0 {
		b.Slacker.SimplePost(
			ev.Channel,
			"Revision not found.",
			messages.IconDefault,
			true,
		)

		return
	}

	if len((*res)[0].Reviewers) == 0 {
		b.Slacker.SimplePost(
			ev.Channel,
			"Revision has no reviewers.",
			messages.IconDefault,
			true,
		)

		return
	}

	authorPHID := (*res)[0].AuthorPHID
	reviewerMap := map[string]bool{}

	for _, reviewerPHID := range (*res)[0].Reviewers {
		nameRes, err := conn.PHIDQuerySingle(reviewerPHID)
		if err != nil {
			b.Excuse(ev, err)
			return
		}

		if (*nameRes).Type == "PROJ" {
			res, err := conn.ProjectQuery(requests.ProjectQueryRequest{
				PHIDs: []string{reviewerPHID},
			})
			if err != nil {
				b.Excuse(ev, err)
				return
			}
			if proj, ok := res.Data[reviewerPHID]; ok {
				res, err := conn.PHIDQuery(requests.PHIDQueryRequest{
					PHIDs: proj.Members,
				})
				if err != nil {
					b.Excuse(ev, err)
					return
				}
				for _, member := range res {
					if member.PHID != authorPHID {
						reviewerMap["@"+member.Name] = true
					}
				}
			}
		} else {
			reviewerMap["@"+(*nameRes).Name] = true
		}
	}

	userInfo, err := b.slackRTM.GetUserInfo(ev.User)
	if err != nil {
		b.Excuse(ev, err)
		return
	}

	reviewerNames := []string{}
	for name := range reviewerMap {
		reviewerNames = append(reviewerNames, name)
	}

	b.Slacker.SimplePost(
		ev.Channel,
		fmt.Sprintf(
			"*@%s summons %s to review D%s:* %s",
			userInfo.Name,
			strings.Join(reviewerNames, ", "),
			matches[1],
			(*res)[0].URI,
		),
		messages.IconDefault,
		true,
	)
}
