package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
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
		skillPath := "/Users/kartikkannan/Desktop/lightcode/skills"
		fmt.Println("Skill path", skillPath)
		skillFilePath := filepath.Join(skillPath, skillName, "SKILL.md")
		fmt.Println("Skill file path", skillFilePath)
		data, err := os.ReadFile(skillFilePath)
		if err != nil {
			return "Skill not found", err
		}
		skill_files := []string{}
		cmd := exec.Command("ls", skillPath+"/"+skillName)
		output, err := cmd.Output()
		if err != nil {
			return "", err
		}
		for s := range strings.SplitSeq(string(output), "\n") {
			if s != "" {
				skill_files = append(skill_files, skillPath+"/"+skillName+"/"+s)
			}
		}
		re := regexp.MustCompile(`(?s)---.*?---`)
		skill := re.ReplaceAllString(string(data), "")
		skill_file_names := "\n<skill_files>\n" + strings.Join(skill_files, "\n") + "\n</skill_files>"
		skill = "<skill_content name=\"" + skillName + "\">" + skill + skill_file_names + "</skill_content>"
		return skill, nil
	})
}
