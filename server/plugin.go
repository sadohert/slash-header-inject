package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, world7!")
}

const minimumServerVersion = "5.4.0"

func (p *Plugin) checkServerVersion() error {
	serverVersion, err := semver.Parse(p.API.GetServerVersion())
	if err != nil {
		return errors.Wrap(err, "failed to parse server version")
	}

	r := semver.MustParseRange(">=" + minimumServerVersion)
	if !r(serverVersion) {
		return fmt.Errorf("this plugin requires Mattermost v%s or later", minimumServerVersion)
	}

	return nil
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
func (p *Plugin) OnActivate() error {
	if err := p.checkServerVersion(); err != nil {
		return err
	}

	//configuration := p.getConfiguration()

	if err := p.registerCommand(); err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}

// TODO Register the new command against all teams
const CommandTrigger = "custom_slash"

func (p *Plugin) registerCommand() error {
	if err := p.API.RegisterCommand(&model.Command{
		Trigger:      CommandTrigger,
		AutoComplete: true,
		//		AutoCompleteHint: "(true|false)",
		AutoCompleteDesc: "<Insert description of slash command endpoint>",
		DisplayName:      "<Call custom command>",
		Description:      "<Calls our in-house enterprise API>",
	}); err != nil {
		return errors.Wrap(err, "failed to register command")
	}

	return nil
}

// TODO Extract customer header KVs for "OnExecute"
// TODO OnExecute follows - https://stackoverflow.com/a/24455606
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	var err error
	//configuration := p.getConfiguration()
	p.API.LogDebug(
		"Executing command - Extracting team/channel/user",
		"CommandArgs", fmt.Sprintf("%+v", args),
	)
	channel, err := p.API.GetChannel(args.ChannelId)
	if err != nil {
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Failed to retrieve channel for ChannelID %s Channel %s ", args.ChannelId, channel.Name),
		}, nil
	}
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("ChannelID %s, ChannelName %s", args.ChannelId, channel.Name),
	}, nil
}
