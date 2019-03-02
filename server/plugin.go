package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-server/model"
	"github.com/mattermost/mattermost-server/plugin"
	"github.com/pkg/errors"
)

const (
	MaxResponseSize = 1024 * 1024 // Posts can be <100KB at most, so this is likely more than enough
)

type Plugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *Configuration
}

func (p *Plugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	config := p.API.GetPluginConfig()
	fmt.Fprintf(w, "CONFIG ARRAY %+v", config)
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

	configuration := p.getConfiguration()
	p.API.LogDebug(
		"Registering custom slash commands",
		"Command Count", len(configuration.SlashCommands),
	)

	if err := p.registerCommands(configuration); err != nil {
		return errors.Wrap(err, "failed to register commands")
	}

	return nil
}

func (p *Plugin) registerCommands(c *Configuration) error {
	for sc, scConfig := range c.SlashCommands {
		p.API.LogDebug(
			"Custom slash command",
			"Trigger", sc,
		)
		if err := p.API.RegisterCommand(&model.Command{
			Trigger:      sc,
			AutoComplete: scConfig.AutoComplete,
			//		AutoCompleteHint: "(true|false)",
			AutoCompleteDesc: scConfig.AutoCompleteDesc,
			DisplayName:      scConfig.DisplayName,
			Description:      scConfig.Description,
		}); err != nil {
			return errors.Wrap(err, "failed to register command")
		}
	}

	return nil
}

// TODO Extract customer header KVs on configuration change
// TODO Extract customer header KVs for "ExecuteCommand"
func (p *Plugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	// var err error
	configuration := p.getConfiguration()
	p.API.LogDebug(
		"Executing command - Extracting team/channel/user",
		"CommandArgs", fmt.Sprintf("%+v", args),
	)
	channel, err := p.API.GetChannel(args.ChannelId)
	if err != nil {
		p.API.LogError(
			"Unable to find channel",
			"Channel", args.ChannelId,
		)
		return nil, model.NewAppError("Unable to find channel", "slash-header-inject", map[string]interface{}{"Channel": args.ChannelId}, err.Error(), http.StatusInternalServerError)
	}
	team, err := p.API.GetTeam(args.TeamId)
	if err != nil {
		p.API.LogError(
			"Unable to find team",
			"Team", args.TeamId,
		)
		return nil, model.NewAppError("Unable to find team", "slash-header-inject", map[string]interface{}{"Team": args.TeamId}, err.Error(), http.StatusInternalServerError)
	}
	user, err := p.API.GetUser(args.UserId)
	if err != nil {
		p.API.LogError(
			"Unable to find user",
			"User", args.UserId,
		)
		return nil, model.NewAppError("Unable to find user", "slash-header-inject", map[string]interface{}{"User": args.UserId}, err.Error(), http.StatusInternalServerError)
	}

	p.API.LogDebug(
		"Executing command - Parse Command",
	)
	// Extract the command
	commandName := strings.TrimSpace(args.Command[1:])
	trigger := ""
	cmdText := ""
	if len(args.Command) != 0 {
		parts := strings.Split(args.Command, " ")
		trigger = parts[0][1:]
		trigger = strings.ToLower(trigger)
		cmdText = strings.Join(parts[1:], " ")
	}

	p.API.LogDebug(
		"Executing command",
		"Command Name", commandName,
	)
	slashcommand := configuration.SlashCommands[trigger]

	// // TODO Check for valid slash command
	// url := slashcommand.CommandURL

	httpParams := url.Values{}
	httpParams.Set("token", "NOT SUPPORTED")

	httpParams.Set("team_id", args.TeamId)
	httpParams.Set("team_domain", team.Name)

	httpParams.Set("channel_id", args.ChannelId)
	httpParams.Set("channel_name", channel.Name)

	httpParams.Set("user_id", args.UserId)
	httpParams.Set("user_name", user.Username)

	httpParams.Set("command", "/"+trigger)
	httpParams.Set("text", cmdText)

	// httpParams.Set("trigger_id", args.TriggerId)
	// httpParams.Set("trigger_id", args.TriggerId)

	//var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
	//req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	var req *http.Request
	var e error
	if slashcommand.RequestType == "GET" {
		req, e = http.NewRequest(http.MethodGet, slashcommand.CommandURL, nil)
	} else {
		req, e = http.NewRequest(http.MethodPost, slashcommand.CommandURL, strings.NewReader(httpParams.Encode()))
	}

	if e != nil {
		// return cmd, nil, model.NewAppError("command", "api.command.execute_command.failed.app_error", map[string]interface{}{"Trigger": cmd.Trigger}, err.Error(), http.StatusInternalServerError)
		panic(e)
	}
	for header, value := range slashcommand.CustomHTTPHeaders {
		req.Header.Set(header, value)
	}

	if slashcommand.RequestType == "GET" {
		if req.URL.RawQuery != "" {
			req.URL.RawQuery += "&"
		}
		req.URL.RawQuery += httpParams.Encode()
	}

	req.Header.Set("Accept", "application/json")
	// TODO - No Token right now req.Header.Set("Authorization", "Token "+slashcommand.Token)
	if slashcommand.RequestType == "POST" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	client := &http.Client{}
	resp, e := client.Do(req)
	if e != nil {
		p.API.LogError(
			"Custom Slash Command http call failed",
			"Command Name", trigger,
			"error", e,
		)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Slash Command Failed - Endpoint Error"),
		}, nil

	}
	defer resp.Body.Close()

	body := io.LimitReader(resp.Body, MaxResponseSize)

	if resp.StatusCode != http.StatusOK {
		// Ignore the error below because the resulting string will just be the empty string if bodyBytes is nil
		bodyBytes, _ := ioutil.ReadAll(body)
		p.API.LogError(
			"Remote server returned failed status",
			"Command Name", trigger,
			"Status", resp.Status,
			"body", string(bodyBytes),
		)
		return &model.CommandResponse{
			ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
			Text:         fmt.Sprintf("Slash Command Failed - Remote Server Error"),
		}, nil
	}

	response, e := model.CommandResponseFromHTTPBody(resp.Header.Get("Content-Type"), body)
	if e != nil {
		return nil, model.NewAppError("Slash Command Failed - Remote Server Error", "slash-header-inject", map[string]interface{}{"Trigger": trigger}, err.Error(), http.StatusInternalServerError)
	} else if response == nil {
		return nil, model.NewAppError("Slash Command Failed - Remote Server Error", "slash-header-inject", map[string]interface{}{"Trigger": trigger}, err.Error(), http.StatusInternalServerError)
	}

	return response, nil

	// return &model.CommandResponse{
	// 	ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
	// 	Text:         fmt.Sprintf(string(body)),
	// }, nil

}

// TODO OnExecute follows - https://stackoverflow.com/a/24455606
