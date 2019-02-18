package main

import (
	"fmt"
	"io/ioutil"
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
		//		"Command Count", len(configuration.slashcommands),
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
	//configuration := p.getConfiguration()

	url := "https://d9b60eb6-42a3-47d5-8032-540a066977ef.mock.pstmn.io/test_path"
	fmt.Println("URL:>", url)

	//var jsonStr = []byte(`{"title":"Buy cheese and bread for breakfast."}`)
	//req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Set("X-Custom-Header", "myvalue")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
	return &model.CommandResponse{
		ResponseType: model.COMMAND_RESPONSE_TYPE_EPHEMERAL,
		Text:         fmt.Sprintf("Custom Command Response: " + string(body)),
	}, nil

}

// TODO OnExecute follows - https://stackoverflow.com/a/24455606
