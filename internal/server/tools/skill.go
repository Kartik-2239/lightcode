package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

func init() {
	Register("skill", ToolDef{
		Name:        "skill",
		Description: "Load a skill from the available skills using skill name",
		Params: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"skillName": map[string]string{
					"type":        "string",
					"description": "The name of the skill to load",
				},
			},
			"required": []string{"skillName"},
		},
	}, func(args map[string]any) (string, error) {
		skillName, ok := args["skillName"].(string)
		if !ok {
			return "", nil
		}
		skillPath := os.Getenv("SKILL_PATH")
		skillFilePath := filepath.Join(skillPath, skillName, "SKILL.md")
		data, err := os.ReadFile(skillFilePath)
		if err != nil {
			return "", err
		}
		cmd := exec.Command("ls", skillPath)
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		files := strings.Split(string(output), "\n")
		skill_files := []string{}
		if len(files) > 0 {
			for _, file := range files {
				cmd := exec.Command("ls", skillPath+"/"+file)
				output, err := cmd.Output()
				if err != nil {
					return "", err
				}
				skill_files = append(skill_files, string(output))
			}
		}
		re := regexp.MustCompile(`(?s)---.*?---`)
		skill := re.ReplaceAllString(string(data), "")
		skill_file_names := "<skill_files>" + strings.Join(skill_files, "\n") + "</skill_files>"
		skill = "<skill_content name=\"" + skillName + "\">" + skill + skill_file_names + "</skill_content>"
		return skill, nil
	})
}
