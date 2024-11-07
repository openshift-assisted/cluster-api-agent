package ignition

import (
	"encoding/json"

	config_types "github.com/coreos/ignition/v2/config/v3_1/types"
)

func GetIgnitionConfigOverrides(files ...config_types.File) (string, error) {
	//[Unit]\nRequires=afterburn.service\nAfter=afterburn.service\n\n
	//contents := `[Service]\nType=oneshot\nEnvironmentFile=/run/metadata/afterburn\nExecStart=/usr/bin/env\n\n[Install]\nWantedBy=multi-user.target`

	//contents := `[Unit]\nRequires=afterburn.service\nAfter=afterburn.service\n\n[Service]\nType=oneshot\nExecStart=/usr/bin/env > /usr/share/myenvs\n\n[Install]\nWantedBy=multi-user.target`
	//enabled := true
	config := config_types.Config{
		Ignition: config_types.Ignition{
			Version: "3.1.0",
		},
		Storage: config_types.Storage{
			Files: files,
		},
		/*		Systemd: config_types.Systemd{
					Units: []config_types.Unit{{
						Contents: &contents,
						Enabled:  &enabled,
						Name:     "dump-envs.service",
					}},
				},
		*/
	}

	ignition, err := json.Marshal(config)
	if err != nil {
		return "", err
	}
	return string(ignition), nil
}

func CreateIgnitionFile(path, user, content string, mode int, overwrite bool) config_types.File {
	return config_types.File{
		Node: config_types.Node{
			Path:      path,
			Overwrite: &overwrite,
			User:      config_types.NodeUser{Name: &user},
		},
		FileEmbedded1: config_types.FileEmbedded1{
			Append: []config_types.Resource{},
			Contents: config_types.Resource{
				Source: &content,
			},
			Mode: &mode,
		},
	}
}
