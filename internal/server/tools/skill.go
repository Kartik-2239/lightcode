package tools

import (
	"fmt"
	"os"
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
	}, func(ctx ToolContext, args map[string]any) (string, error) {
		skillName, ok := args["skillName"].(string)
		if !ok {
			return "", nil
		}
		skillPath := os.Getenv("SKILL_PATH")
		fmt.Println("Skill path", skillPath)
		skillFilePath := filepath.Join(skillPath, skillName, "SKILL.md")
		fmt.Println("Skill file path", skillFilePath)
		data, err := os.ReadFile(skillFilePath)
		if err != nil {
			return "Skill not found", err
		}
		skillDir := filepath.Join(skillPath, skillName)
		entries, err := os.ReadDir(skillDir)
		if err != nil {
			return "", err
		}
		skill_files := []string{}
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}
			filePath := filepath.Join(skillDir, entry.Name())
			fileData, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			skill_files = append(skill_files, "<file path=\""+filePath+"\">\n"+string(fileData)+"\n</file>")
		}
		re := regexp.MustCompile(`(?s)---.*?---`)
		skill := re.ReplaceAllString(string(data), "")
		skillFilesBlock := "\n<skill_files>\n" + strings.Join(skill_files, "\n") + "\n</skill_files>"
		skill = "<skill_content name=\"" + skillName + "\">" + skill + skillFilesBlock + "</skill_content>"
		return skill, nil
	})
}
