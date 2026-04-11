package commands

import (
	"fmt"
	"io"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const (
	bashCompletionFunc = `
__athena_list_apps() {
	local -a athena_out
	if athena_out=($(athena app list --output name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_list_app_history() {
	local app=$1
	local -a athena_out
	if athena_out=($(athena app history $app --output id 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_app_rollback() {
	local -a command
	for comp_word in "${COMP_WORDS[@]}"; do
		if [[ $comp_word =~ ^-.*$ ]]; then
			continue
		fi
		command+=($comp_word)
	done

	# fourth arg is app (if present): e.g.- athena app rollback guestbook
	local app=${command[3]}
	local id=${command[4]}
	if [[ -z $app || $app == $cur ]]; then
		__athena_list_apps
	elif [[ -z $id || $id == $cur ]]; then
		__athena_list_app_history $app
	fi
}

__athena_list_servers() {
	local -a athena_out
	if athena_out=($(athena cluster list --output server 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_list_repos() {
	local -a athena_out
	if athena_out=($(athena repo list --output url 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_list_projects() {
	local -a athena_out
	if athena_out=($(athena proj list --output name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_list_namespaces() {
	local -a athena_out
	if athena_out=($(kubectl get namespaces --no-headers 2>/dev/null | cut -f1 -d' ' 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_proj_server_namespace() {
	local -a command
	for comp_word in "${COMP_WORDS[@]}"; do
		if [[ $comp_word =~ ^-.*$ ]]; then
			continue
		fi
		command+=($comp_word)
	done

	# expect something like this: athena proj add-destination PROJECT SERVER NAMESPACE
	local project=${command[3]}
	local server=${command[4]}
	local namespace=${command[5]}
	if [[ -z $project || $project == $cur ]]; then
		__athena_list_projects
	elif [[ -z $server || $server == $cur ]]; then
		__athena_list_servers
	elif [[ -z $namespace || $namespace == $cur ]]; then
		__athena_list_namespaces
	fi
}

__athena_list_project_role() {
	local project="$1"
	local -a athena_out
	if athena_out=($(athena proj role list "$project" --output=name 2>/dev/null)); then
		COMPREPLY+=( $( compgen -W "${athena_out[*]}" -- "$cur" ) )
	fi
}

__athena_proj_role(){
	local -a command
	for comp_word in "${COMP_WORDS[@]}"; do
		if [[ $comp_word =~ ^-.*$ ]]; then
			continue
		fi
		command+=($comp_word)
	done

	# expect something like this: athena proj role add-policy PROJECT ROLE-NAME
	local project=${command[4]}
	local role=${command[5]}
	if [[ -z $project || $project == $cur ]]; then
		__athena_list_projects
	elif [[ -z $role || $role == $cur ]]; then
		__athena_list_project_role $project
	fi
}

__athena_custom_func() {
	case ${last_command} in
		athena_app_delete | \
		athena_app_diff | \
		athena_app_edit | \
		athena_app_get | \
		athena_app_history | \
		athena_app_manifests | \
		athena_app_patch-resource | \
		athena_app_set | \
		athena_app_sync | \
		athena_app_terminate-op | \
		athena_app_unset | \
		athena_app_wait | \
		athena_app_create)
			__athena_list_apps
			return
			;;
		athena_app_rollback)
			__athena_app_rollback
			return
			;;
		athena_cluster_get | \
		athena_cluster_rm | \
		athena_cluster_set | \
		athena_login | \
		athena_cluster_add)
			__athena_list_servers
			return
			;;
		athena_repo_rm | \
		athena_repo_add)
			__athena_list_repos
			return
			;;
		athena_proj_add-destination | \
		athena_proj_remove-destination)
			__athena_proj_server_namespace
			return
			;;
		athena_proj_add-source | \
		athena_proj_remove-source | \
		athena_proj_allow-cluster-resource | \
		athena_proj_allow-namespace-resource | \
		athena_proj_deny-cluster-resource | \
		athena_proj_deny-namespace-resource | \
		athena_proj_delete | \
		athena_proj_edit | \
		athena_proj_get | \
		athena_proj_set | \
		athena_proj_role_list)
			__athena_list_projects
			return
			;;
		athena_proj_role_remove-policy | \
		athena_proj_role_add-policy | \
		athena_proj_role_create | \
		athena_proj_role_delete | \
		athena_proj_role_get | \
		athena_proj_role_create-token | \
		athena_proj_role_delete-token)
			__athena_proj_role
			return
			;;
		*)
			;;
	esac
}
	`
)

func NewCompletionCommand() *cobra.Command {
	command := &cobra.Command{
		Use:   "completion SHELL",
		Short: "output shell completion code for the specified shell (bash, zsh or fish)",
		Long: `Write bash, zsh or fish shell completion code to standard output.

For bash, ensure you have bash completions installed and enabled.
To access completions in your current shell, run
$ source <(athena completion bash)
Alternatively, write it to a file and source in .bash_profile

For zsh, add the following to your ~/.zshrc file:
source <(athena completion zsh)
compdef _athena athena

Optionally, also add the following, in case you are getting errors involving compdef & compinit such as command not found: compdef:
autoload -Uz compinit
compinit
`,
		Example: `# For bash
$ source <(athena completion bash)

# For zsh
$ athena completion zsh > _athena
$ source _athena

# For fish
$ athena completion fish > ~/.config/fish/completions/athena.fish
$ source ~/.config/fish/completions/athena.fish

# For powershell
$ mkdir -Force "$HOME\Documents\PowerShell" | Out-Null
$ athena completion powershell > $HOME\Documents\PowerShell\athena_completion.ps1

Add the following lines to your powershell profile

$ # Athena tab completion
if (Test-Path "$HOME\Documents\PowerShell\athena_completion.ps1") {
    . "$HOME\Documents\PowerShell\athena_completion.ps1"
}

Then reload your profile
$ . $PROFILE
`,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) != 1 {
				cmd.HelpFunc()(cmd, args)
				os.Exit(1)
			}
			shell := args[0]
			rootCommand := NewCommand()
			rootCommand.BashCompletionFunction = bashCompletionFunc
			availableCompletions := map[string]func(out io.Writer, cmd *cobra.Command) error{
				"bash":       runCompletionBash,
				"zsh":        runCompletionZsh,
				"fish":       runCompletionFish,
				"powershell": runCompletionPowershell,
			}
			completion, ok := availableCompletions[shell]
			if !ok {
				fmt.Printf("Invalid shell '%s'. The supported shells are bash, zsh and fish.\n", shell)
				os.Exit(1)
			}
			if err := completion(os.Stdout, rootCommand); err != nil {
				log.Fatal(err)
			}
		},
	}

	return command
}

func runCompletionBash(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenBashCompletion(out)
}

func runCompletionZsh(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenZshCompletion(out)
}

func runCompletionFish(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenFishCompletion(out, true)
}

func runCompletionPowershell(out io.Writer, cmd *cobra.Command) error {
	return cmd.GenPowerShellCompletionWithDesc(out)
}
