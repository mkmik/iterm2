package iterm2

import (
	"fmt"
	"strconv"

	"github.com/mkmik/iterm2/api"
	"github.com/mkmik/iterm2/client"
)

// Window represents an iTerm2 Window
type Window interface {
	SetTitle(s string) error
	CreateTab() (Tab, error)
	ListTabs() ([]Tab, error)
	Activate() error
	GetVariable(name string) (string, error)
	SetVariable(name, value string) error
}

type window struct {
	c       *client.Client
	id      string
	session string
}

func (w *window) CreateTab() (Tab, error) {
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{
				WindowId: str(w.id),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create tab for window %q: %w", w.id, err)
	}
	ctr := resp.GetCreateTabResponse()
	if ctr.GetStatus() != api.CreateTabResponse_OK {
		return nil, fmt.Errorf("unexpected tab status: %s", ctr.GetStatus())
	}
	return &tab{
		c:        w.c,
		id:       strconv.Itoa(int(ctr.GetTabId())),
		windowID: w.id,
	}, nil
}

func (w *window) ListTabs() ([]Tab, error) {
	list := []Tab{}
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not list sessions: %w", err)
	}
	for _, window := range resp.GetListSessionsResponse().GetWindows() {
		if window.GetWindowId() != w.id {
			continue
		}
		for _, t := range window.GetTabs() {
			list = append(list, &tab{
				c:        w.c,
				id:       t.GetTabId(),
				windowID: w.id,
			})
		}
	}
	return list, nil
}

func (w *window) SetTitle(s string) error {
	_, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: str(fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)),
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &w.id,
					},
				},
			},
		},
	})
	return err
}

func (w *window) Activate() error {
	_, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{ActivateRequest: &api.ActivateRequest{
			Identifier:       &api.ActivateRequest_WindowId{WindowId: w.id},
			OrderWindowFront: b(true),
		}},
	})
	return err
}

func (w *window) GetVariable(name string) (string, error) {
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_VariableRequest{
			VariableRequest: &api.VariableRequest{
				Scope: &api.VariableRequest_WindowId{WindowId: w.id},
				Get:   []string{name},
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("error getting variable %q for window %q: %w", name, w.id, err)
	}
	varResp := resp.GetVariableResponse()
	if status := varResp.GetStatus(); status != api.VariableResponse_OK {
		return "", fmt.Errorf("unexpected status getting variable %q: %s", name, status)
	}
	values := varResp.GetValues()
	if len(values) == 0 {
		return "", nil
	}
	return values[0], nil
}

func (w *window) SetVariable(name, value string) error {
	resp, err := w.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_VariableRequest{
			VariableRequest: &api.VariableRequest{
				Scope: &api.VariableRequest_WindowId{WindowId: w.id},
				Set: []*api.VariableRequest_Set{{
					Name:  &name,
					Value: &value,
				}},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error setting variable %q for window %q: %w", name, w.id, err)
	}
	if status := resp.GetVariableResponse().GetStatus(); status != api.VariableResponse_OK {
		return fmt.Errorf("unexpected status setting variable %q: %s", name, status)
	}
	return nil
}
