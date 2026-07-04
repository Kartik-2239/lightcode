package tools

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Kartik-2239/lightcode/internal/server/config"
)

func Skill(ctx ToolContext, args map[string]any) (ToolResponse, error) {
	skillName, ok := args["skillName"].(string)
	if !ok {
		return ToolResponse{Content: "Error: skillName is required and must be a string", Printable: "Error: skillName is required and must be a string"}, nil
	}
	skillPath := config.SkillsPath()
	if skillPath == "" {
		return ToolResponse{Content: "Skill path not found", Printable: "Skill path not found"}, nil
	}
	skillFilePath := filepath.Join(skillPath, skillName, "SKILL.md")
	data, err := os.ReadFile(skillFilePath)
	if err != nil {
		return ToolResponse{Content: "Skill not found", Printable: "Skill not found"}, err
	}
	skillDir := filepath.Join(skillPath, skillName)
	entries, err := os.ReadDir(skillDir)
	if err != nil {
		return ToolResponse{Content: "Error: " + err.Error(), Printable: "Error: " + err.Error()}, err
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
	return ToolResponse{Content: skill, Printable: skill}, nil
}

func init() {
	Prompt := `Load a skill from the available skills using skill name
	
- Skills are lazy loaded prompts and definition of how to use/implement something.
- use this tool to load skills when required`
	Register("skill", ToolDef{
		Name:        "skill",
		Description: Prompt,
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
	}, Skill)
}
