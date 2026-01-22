package iterm2

import (
	"encoding/json"
	"fmt"

	"github.com/mkmik/iterm2/api"
	"github.com/mkmik/iterm2/client"
)

// Tab abstracts an iTerm2 window tab
type Tab interface {
	SetTitle(string) error
	ListSessions() ([]Session, error)
	GetVariable(name string, value any) error
	SetVariable(name string, value any) error
}

type tab struct {
	c        *client.Client
	id       string
	windowID string
}

func (t *tab) SetTitle(s string) error {
	_, err := t.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: str(fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)),
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &t.id,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("could not call set_title: %w", err)
	}
	return nil
}

func (t *tab) ListSessions() ([]Session, error) {
	list := []Session{}
	resp, err := t.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error listing sessions for tab %q: %w", t.id, err)
	}
	lsr := resp.GetListSessionsResponse()
	for _, window := range lsr.GetWindows() {
		if window.GetWindowId() != t.windowID {
			continue
		}
		for _, wt := range window.GetTabs() {
			if wt.GetTabId() != t.id {
				continue
			}
			for _, link := range wt.GetRoot().GetLinks() {
				list = append(list, &session{
					c:  t.c,
					id: link.GetSession().GetUniqueIdentifier(),
				})
			}
		}
	}
	return list, nil
}

func (t *tab) GetVariable(name string, value any) error {
	resp, err := t.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_VariableRequest{
			VariableRequest: &api.VariableRequest{
				Scope: &api.VariableRequest_TabId{TabId: t.id},
				Get:   []string{name},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error getting variable %q: %w", name, err)
	}
	varResp := resp.GetVariableResponse()
	if status := varResp.GetStatus(); status != api.VariableResponse_OK {
		return fmt.Errorf("unexpected status getting variable %q: %s", name, status)
	}
	values := varResp.GetValues()
	if len(values) == 0 || values[0] == "null" {
		return nil
	}
	if err := json.Unmarshal([]byte(values[0]), value); err != nil {
		return fmt.Errorf("error unmarshaling variable %q: %w", name, err)
	}
	return nil
}

func (t *tab) SetVariable(name string, value any) error {
	jsonBytes, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("error marshaling value for variable %q: %w", name, err)
	}
	jsonStr := string(jsonBytes)

	resp, err := t.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_VariableRequest{
			VariableRequest: &api.VariableRequest{
				Scope: &api.VariableRequest_TabId{TabId: t.id},
				Set: []*api.VariableRequest_Set{{
					Name:  &name,
					Value: &jsonStr,
				}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error setting variable %q: %w", name, err)
	}
	if status := resp.GetVariableResponse().GetStatus(); status != api.VariableResponse_OK {
		return fmt.Errorf("unexpected status setting variable %q: %s", name, status)
	}
	return nil
}
