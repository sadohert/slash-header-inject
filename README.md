# Custom HTTP Header Slash Plugin 

This plugin allows Mattermost administrators to create custom slash commands (similar to what can already be done through the UI) but with a configurable list of custom HTTP headers added to the `GET` or `POST` calls

## Installation

1. Go to the releases tab of this Github repository and download the latest release.
2. Upload this file in the Mattermost **System Console > Plugins > Management** page to install the plugin. To learn more about how to upload a plugin, [see the documentation](https://docs.mattermost.com/administration/plugins.html#plugin-uploads).
3. Modify your `config.json` file (`PluginSettings` section) to include a map of custom slash commands and their desired custom headers as shown below.

## Custom Slash Command Definition

Follow the same set of configurable properties as  used in the [built-in custom slash commands](https://docs.mattermost.com/developer/slash-commands.html#custom-slash-command).  Note, commands must have an alphanumeric name.

See example below:

```
        "Plugins": {
            "slash-header-inject": {
                "slashcommands": {
                    "weather": {
                        "autocomplete": true,
                        "autocompletedesc": "Display the weather",
                        "commandurl": "http://myweatherservice.com",
                        "customhttpheaders": {
                            "x-headerx": "X-Value",
                            "x-headery": "Y-Value",
                        },
                        "description": "Weather slash command descriptions",
                        "displayname": "Weather slash command  display name",
                        "requesttype": "GET"
                    },
                    "stock_ticker": {
                        "autocomplete": true,
                        "autocompletedesc": "test_config_array2 autocomplete description",
                        "commandurl": "http://localhost:3000",
                        "customhttpheaders": {
                            "x-mattermost-slash-header": "TEST_HEADER_VALUE"
                        },
                        "description": "Description goes here",
                        "displayname": "display name goes here",
                        "requesttype": "POST"
                    }
                }
            }
        }

```