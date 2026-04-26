// Package cli 提供 Cobra 命令行输入适配层。
package cli

import (
	"context"
	"fmt"
	"io"

	"github.com/finger/skill-down/internal/skilldown/app"
	"github.com/finger/skill-down/internal/skilldown/tui"
	"github.com/spf13/cobra"
)

var runTUI = tui.Run

// Execute 启动 CLI 入口。
func Execute(ctx context.Context, args []string, svc app.Service) int {
	return execute(ctx, args, svc, nil, nil)
}

func execute(ctx context.Context, args []string, svc app.Service, stdout io.Writer, stderr io.Writer) int {
	root := newRootCommand(ctx, svc)
	root.SetArgs(args)
	if stdout != nil {
		root.SetOut(stdout)
	}
	if stderr != nil {
		root.SetErr(stderr)
	}

	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintln(root.ErrOrStderr(), err)
		return 1
	}
	return 0
}

func newRootCommand(ctx context.Context, svc app.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:           "skilldown",
		Short:         "发现和安装远程仓库中的 skills",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	cmd.AddCommand(newRegistryCommand(ctx, svc))
	cmd.AddCommand(newSearchCommand(ctx, svc))
	cmd.AddCommand(newInstallCommand(ctx, svc))
	cmd.AddCommand(newBrowseCommand(ctx, svc))
	return cmd
}

func newBrowseCommand(ctx context.Context, svc app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "browse [repo]",
		Short: "交互式浏览和安装 skill",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runTUI(ctx, svc, optionalArg(args))
		},
	}
}

func newRegistryCommand(ctx context.Context, svc app.Service) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "registry",
		Short: "管理已注册的 skill 仓库",
	}
	cmd.AddCommand(newRegistryAddCommand(ctx, svc))
	cmd.AddCommand(newRegistryListCommand(ctx, svc))
	cmd.AddCommand(newRegistryRemoveCommand(ctx, svc))
	return cmd
}

func newRegistryAddCommand(ctx context.Context, svc app.Service) *cobra.Command {
	var name string
	var skillDirPath string

	cmd := &cobra.Command{
		Use:   "add <repo>",
		Short: "添加 skill 仓库",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			return svc.AddRepository(ctx, app.AddRepositoryRequest{
				Name:         name,
				URL:          args[0],
				SkillDirPath: skillDirPath,
			})
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "registry 名称")
	cmd.Flags().StringVar(&skillDirPath, "skill-dir", "", "仓库内 skill 目录")
	return cmd
}

func newRegistryListCommand(ctx context.Context, svc app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "列出 skill 仓库",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repositories, err := svc.ListRepositories(ctx)
			if err != nil {
				return err
			}
			for _, repository := range repositories {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\n", repository.Name, repository.URL, repository.SkillDir)
			}
			return nil
		},
	}
}

func newRegistryRemoveCommand(ctx context.Context, svc app.Service) *cobra.Command {
	return &cobra.Command{
		Use:   "remove <name>",
		Short: "删除 skill 仓库",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return svc.RemoveRepository(ctx, app.RemoveRepositoryRequest{Name: args[0]})
		},
	}
}

func newSearchCommand(ctx context.Context, svc app.Service) *cobra.Command {
	var skillDirPath string

	cmd := &cobra.Command{
		Use:   "search [repo]",
		Short: "搜索 skill",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := optionalArg(args)
			result, err := svc.Search(ctx, app.SearchRequest{
				RepoURL:      repoURL,
				SkillDirPath: skillDirPath,
			})
			if err != nil {
				return err
			}
			printSearchResult(cmd.OutOrStdout(), result)
			return nil
		},
	}
	cmd.Flags().StringVar(&skillDirPath, "skill-dir", "", "仓库内 skill 目录")
	return cmd
}

func newInstallCommand(ctx context.Context, svc app.Service) *cobra.Command {
	var skillName string
	var skillDirPath string
	var targetRoot string
	var force bool

	cmd := &cobra.Command{
		Use:   "install [repo]",
		Short: "安装 skill",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repoURL := optionalArg(args)
			result, err := svc.Install(ctx, app.InstallRequest{
				RepoURL:      repoURL,
				SkillName:    skillName,
				SkillDirPath: skillDirPath,
				TargetRoot:   targetRoot,
				Force:        force,
			})
			printInstallResult(cmd.OutOrStdout(), result)
			return err
		},
	}
	cmd.Flags().StringVar(&skillName, "skill", "", "要安装的 skill 名称")
	cmd.Flags().StringVar(&skillDirPath, "skill-dir", "", "仓库内 skill 目录")
	cmd.Flags().StringVar(&targetRoot, "to", "", "安装根目录")
	cmd.Flags().BoolVar(&force, "force", false, "覆盖已存在目标目录")
	return cmd
}

func optionalArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}

func printSearchResult(out io.Writer, result app.SearchResult) {
	for _, group := range result.Repositories {
		_, _ = fmt.Fprintf(out, "%s\t%s\n", group.Repository.Name, group.Repository.URL)
		for _, found := range group.Skills {
			_, _ = fmt.Fprintf(out, "- %s\t%s\t%s\n", found.Name, found.Path, found.Description)
		}
	}
}

func printInstallResult(out io.Writer, result app.InstallResult) {
	for _, item := range result.Results {
		if item.Copy.Err != nil {
			_, _ = fmt.Fprintf(out, "failed\t%s\t%s\t%v\n", item.Skill.Name, item.Copy.TargetDir, item.Copy.Err)
			continue
		}
		_, _ = fmt.Fprintf(out, "%s\t%s\t%s\n", item.Copy.Status, item.Skill.Name, item.Copy.TargetDir)
	}
}
