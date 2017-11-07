package pivotal

import (
	"fmt"

	"github.com/paroxp/paas-rubbernecker/pkg/rubbernecker"
	pt "github.com/salsita/go-pivotaltracker/v5/pivotal"
)

// Tracker will be responsible for acting as the story resource returned
// by the API.
type Tracker struct {
	client    *pt.Client
	projectID int
	stories   []*story
	members   []*membership
}

// New will compose a Tracker struct ready to use by the rubbernecker.
func New(projectID int, token string) (*Tracker, error) {
	return &Tracker{
		client:    pt.NewClient(token),
		projectID: projectID,
	}, nil
}

// FetchCards will fetch the stories from PivotalTracker.
func (t *Tracker) FetchCards(status rubbernecker.Status) error {
	fields := "owner_ids,blockers,transitions,current_state,labels,name,url,created_at"
	path := fmt.Sprintf("projects/%d/stories?fields=%s&filter=state:%s", t.projectID, fields, composeState(status))

	req, err := t.client.NewRequest("GET", path, nil)
	if err != nil {
		return err
	}

	t.stories = []*story{}
	_, err = t.client.Do(req, &t.stories)
	if err != nil {
		return err
	}

	return nil
}

// FlattenStories function will take what we have so far and convert it into the
// rubbernecker standard.
func (t *Tracker) FlattenStories() (*rubbernecker.Cards, error) {
	if len(t.stories) == 0 {
		return nil, fmt.Errorf("pivotal extension: no stories to be flattened")
	}

	stories := rubbernecker.Cards{}

	for _, s := range t.stories {
		stickers := labelsToStickers(s.Labels)

		if len(s.Blockers) > 0 && !inSlice(stickers, "blocked") {
			stickers = append(stickers, "blocked")
		}

		assignees := rubbernecker.Members{}

		for _, id := range s.OwnerIds {
			assignees[id] = &rubbernecker.Member{ID: id}
		}

		stories = append(stories, &rubbernecker.Card{
			ID:        s.ID,
			Assignees: &assignees,
			Elapsed:   calculateInState(s.Transitions, s.State),
			Status:    convertState(s.State),
			Stickers:  stickers,
			Title:     s.Name,
			URL:       s.URL,
		})
	}

	return &stories, nil
}
