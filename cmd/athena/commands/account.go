package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	timeutil "github.com/useryege/athena/pkg/time"
	"golang.org/x/term"
	"sigs.k8s.io/yaml"

	"github.com/useryege/athena/util/rbac"

	"github.com/useryege/athena/cmd/athena/commands/headless"
	"github.com/useryege/athena/cmd/athena/commands/utils"
	athenaclient "github.com/useryege/athena/pkg/apiclient"
	accountpkg "github.com/useryege/athena/pkg/apiclient/account"
	"github.com/useryege/athena/pkg/apiclient/session"
	"github.com/useryege/athena/util/cli"
	"github.com/useryege/athena/util/errors"
	utilio "github.com/useryege/athena/util/io"
	"github.com/useryege/athena/util/localconfig"
	sessionutil "github.com/useryege/athena/util/session"
	"github.com/useryege/athena/util/templates"
)

func NewAccountCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	command := &cobra.Command{
		Use:   "account",
		Short: "Manage account settings",
		Example: templates.Examples(`
			# List accounts
			athena account list

			# Update the current user's password
			athena account update-password

			# Can I sync any app?
			athena account can-i sync applications '*'

			# Get User information
			athena account get-user-info
		`),
		Run: func(c *cobra.Command, args []string) {
			c.HelpFunc()(c, args)
			os.Exit(1)
		},
	}
	command.AddCommand(NewAccountUpdatePasswordCommand(clientOpts))
	command.AddCommand(NewAccountGetUserInfoCommand(clientOpts))
	command.AddCommand(NewAccountCanICommand(clientOpts))
	command.AddCommand(NewAccountListCommand(clientOpts))
	command.AddCommand(NewAccountGenerateTokenCommand(clientOpts))
	command.AddCommand(NewAccountGetCommand(clientOpts))
	command.AddCommand(NewAccountDeleteTokenCommand(clientOpts))
	command.AddCommand(NewBcryptCmd())
	return command
}

func NewAccountUpdatePasswordCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	var (
		account         string
		currentPassword string
		newPassword     string
	)
	command := &cobra.Command{
		Use:   "update-password",
		Short: "Update an account's password",
		Long: `
This command can be used to update the password of the currently logged on
user, or an arbitrary local user account when the currently logged on user
has appropriate RBAC permissions to change other accounts.
`,
		Example: `
	# Update the current user's password
	athena account update-password

	# Update the password for user foobar
	athena account update-password --account foobar
`,
		Run: func(c *cobra.Command, args []string) {
			ctx := c.Context()

			if len(args) != 0 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			acdClient := headless.NewClientOrDie(clientOpts, c)
			conn, usrIf := acdClient.NewAccountClientOrDie()
			defer utilio.Close(conn)

			userInfo := getCurrentAccount(ctx, acdClient)

			if userInfo.Iss == sessionutil.SessionManagerClaimsIssuer && currentPassword == "" {
				fmt.Printf("*** Enter password of currently logged in user (%s): ", userInfo.Username)
				password, err := term.ReadPassword(int(os.Stdin.Fd()))
				errors.CheckError(err)
				currentPassword = string(password)
				fmt.Print("\n")
			}

			if account == "" {
				account = userInfo.Username
			}

			if newPassword == "" {
				var err error
				newPassword, err = cli.ReadAndConfirmPassword(account)
				errors.CheckError(err)
			}

			updatePasswordRequest := accountpkg.UpdatePasswordRequest{
				NewPassword:     newPassword,
				CurrentPassword: currentPassword,
				Name:            account,
			}

			_, err := usrIf.UpdatePassword(ctx, &updatePasswordRequest)
			errors.CheckError(err)
			fmt.Printf("Password updated\n")

			if account == "" || account == userInfo.Username {
				// Get a new JWT token after updating the password
				localCfg, err := localconfig.ReadLocalConfig(clientOpts.ConfigPath)
				errors.CheckError(err)
				configCtx, err := localCfg.ResolveContext(clientOpts.Context)
				errors.CheckError(err)
				claims, err := configCtx.User.Claims()
				errors.CheckError(err)
				tokenString := passwordLogin(ctx, acdClient, localconfig.GetUsername(claims.Subject), newPassword)
				localCfg.UpsertUser(localconfig.User{
					Name:      localCfg.CurrentContext,
					AuthToken: tokenString,
				})
				err = localconfig.WriteLocalConfig(*localCfg, clientOpts.ConfigPath)
				errors.CheckError(err)
				fmt.Printf("Context '%s' updated\n", localCfg.CurrentContext)
			}
		},
	}

	command.Flags().StringVar(&currentPassword, "current-password", "", "Password of the currently logged on user")
	command.Flags().StringVar(&newPassword, "new-password", "", "New password you want to update to")
	command.Flags().StringVar(&account, "account", "", "An account name that should be updated. Defaults to current user account")
	return command
}

func NewAccountGetUserInfoCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	var output string
	command := &cobra.Command{
		Use:     "get-user-info",
		Short:   "Get user info",
		Aliases: []string{"whoami"},
		Example: templates.Examples(`
			# Get User information for the currently logged-in user (see 'athena login')
			athena account get-user-info

			# Get User information in yaml format
			athena account get-user-info -o yaml
		`),
		Run: func(c *cobra.Command, args []string) {
			ctx := c.Context()

			if len(args) != 0 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}

			conn, client := headless.NewClientOrDie(clientOpts, c).NewSessionClientOrDie()
			defer utilio.Close(conn)

			response, err := client.GetUserInfo(ctx, &session.GetUserInfoRequest{})
			errors.CheckError(err)

			switch output {
			case "yaml":
				yamlBytes, err := yaml.Marshal(response)
				errors.CheckError(err)
				fmt.Println(string(yamlBytes))
			case "json":
				jsonBytes, err := json.MarshalIndent(response, "", "  ")
				errors.CheckError(err)
				fmt.Println(string(jsonBytes))
			case "":
				fmt.Printf("Logged In: %v\n", response.LoggedIn)
				if response.LoggedIn {
					fmt.Printf("Username: %s\n", response.Username)
					fmt.Printf("Issuer: %s\n", response.Iss)
					fmt.Printf("Groups: %v\n", strings.Join(response.Groups, ","))
				}
			default:
				log.Fatalf("Unknown output format: %s", output)
			}
		},
	}
	command.Flags().StringVarP(&output, "output", "o", "", "Output format. One of: yaml, json")
	return command
}

func NewAccountCanICommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "can-i ACTION RESOURCE SUBRESOURCE",
		Short: "Can I",
		Example: fmt.Sprintf(`
# Can I sync any app?
athena account can-i sync applications '*'

# Can I update a project?
athena account can-i update projects 'default'

# Can I create a cluster?
athena account can-i create clusters '*'

Actions: %v
Resources: %v
`, rbac.Actions, rbac.Resources),
		Run: func(c *cobra.Command, args []string) {
			ctx := c.Context()

			if len(args) != 3 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}

			conn, client := headless.NewClientOrDie(clientOpts, c).NewAccountClientOrDie()
			defer utilio.Close(conn)

			response, err := client.CanI(ctx, &accountpkg.CanIRequest{
				Action:      args[0],
				Resource:    args[1],
				Subresource: args[2],
			})
			errors.CheckError(err)
			fmt.Println(response.Value)
		},
	}
}

func printAccountNames(accounts []*accountpkg.Account) {
	for _, p := range accounts {
		fmt.Println(p.Name)
	}
}

func printAccountsTable(items []*accountpkg.Account) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "NAME\tENABLED\tCAPABILITIES\n")
	for _, a := range items {
		fmt.Fprintf(w, "%s\t%v\t%s\n", a.Name, a.Enabled, strings.Join(a.Capabilities, ", "))
	}
	_ = w.Flush()
}

func NewAccountListCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	var output string
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List accounts",
		Example: "athena account list",
		Run: func(c *cobra.Command, _ []string) {
			ctx := c.Context()

			conn, client := headless.NewClientOrDie(clientOpts, c).NewAccountClientOrDie()
			defer utilio.Close(conn)

			response, err := client.ListAccounts(ctx, &accountpkg.ListAccountRequest{})

			errors.CheckError(err)
			switch output {
			case "yaml", "json":
				err := PrintResourceList(response.Items, output, false)
				errors.CheckError(err)
			case "name":
				printAccountNames(response.Items)
			case "wide", "":
				printAccountsTable(response.Items)
			default:
				errors.CheckError(fmt.Errorf("unknown output format: %s", output))
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "wide", "Output format. One of: json|yaml|wide|name")
	return cmd
}

func getCurrentAccount(ctx context.Context, clientset athenaclient.Client) session.GetUserInfoResponse {
	conn, client := clientset.NewSessionClientOrDie()
	defer utilio.Close(conn)
	userInfo, err := client.GetUserInfo(ctx, &session.GetUserInfoRequest{})
	errors.CheckError(err)
	return *userInfo
}

func NewAccountGetCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	var (
		output  string
		account string
	)
	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get account details",
		Example: `# Get the currently logged in account details
athena account get

# Get details for an account by name
athena account get --account <account-name>`,
		Run: func(c *cobra.Command, _ []string) {
			ctx := c.Context()

			clientset := headless.NewClientOrDie(clientOpts, c)

			if account == "" {
				account = getCurrentAccount(ctx, clientset).Username
			}

			conn, client := clientset.NewAccountClientOrDie()
			defer utilio.Close(conn)

			acc, err := client.GetAccount(ctx, &accountpkg.GetAccountRequest{Name: account})

			errors.CheckError(err)
			switch output {
			case "yaml", "json":
				err := PrintResourceList(acc, output, true)
				errors.CheckError(err)
			case "name":
				fmt.Println(acc.Name)
			case "wide", "":
				printAccountDetails(acc)
			default:
				errors.CheckError(fmt.Errorf("unknown output format: %s", output))
			}
		},
	}
	cmd.Flags().StringVarP(&output, "output", "o", "wide", "Output format. One of: json|yaml|wide|name")
	cmd.Flags().StringVarP(&account, "account", "a", "", "Account name. Defaults to the current account.")
	return cmd
}

func printAccountDetails(acc *accountpkg.Account) {
	fmt.Printf(printOpFmtStr, "Name:", acc.Name)
	fmt.Printf(printOpFmtStr, "Enabled:", strconv.FormatBool(acc.Enabled))
	fmt.Printf(printOpFmtStr, "Capabilities:", strings.Join(acc.Capabilities, ", "))
	fmt.Println("\nTokens:")
	if len(acc.Tokens) == 0 {
		fmt.Println("NONE")
	} else {
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID\tISSUED AT\tEXPIRING AT\n")
		for _, t := range acc.Tokens {
			expiresAtFormatted := "never"
			if t.ExpiresAt > 0 {
				expiresAt := time.Unix(t.ExpiresAt, 0)
				expiresAtFormatted = expiresAt.Format(time.RFC3339)
				if expiresAt.Before(time.Now()) {
					expiresAtFormatted = expiresAtFormatted + " (expired)"
				}
			}

			fmt.Fprintf(w, "%s\t%s\t%s\n", t.Id, time.Unix(t.IssuedAt, 0).Format(time.RFC3339), expiresAtFormatted)
		}
		_ = w.Flush()
	}
}

func NewAccountGenerateTokenCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	var (
		account   string
		expiresIn string
		id        string
	)
	cmd := &cobra.Command{
		Use:   "generate-token",
		Short: "Generate account token",
		Example: `# Generate token for the currently logged in account
athena account generate-token

# Generate token for the account with the specified name
athena account generate-token --account <account-name>`,
		Run: func(c *cobra.Command, _ []string) {
			ctx := c.Context()

			clientset := headless.NewClientOrDie(clientOpts, c)
			conn, client := clientset.NewAccountClientOrDie()
			defer utilio.Close(conn)
			if account == "" {
				account = getCurrentAccount(ctx, clientset).Username
			}
			expiresIn, err := timeutil.ParseDuration(expiresIn)
			errors.CheckError(err)
			response, err := client.CreateToken(ctx, &accountpkg.CreateTokenRequest{
				Name:      account,
				ExpiresIn: int64(expiresIn.Seconds()),
				Id:        id,
			})
			errors.CheckError(err)
			fmt.Println(response.Token)
		},
	}
	cmd.Flags().StringVarP(&account, "account", "a", "", "Account name. Defaults to the current account.")
	cmd.Flags().StringVarP(&expiresIn, "expires-in", "e", "0s", "Duration before the token will expire. (Default: No expiration)")
	cmd.Flags().StringVar(&id, "id", "", "Optional token id. Fall back to uuid if not value specified.")
	return cmd
}

func NewAccountDeleteTokenCommand(clientOpts *athenaclient.ClientOptions) *cobra.Command {
	var account string
	cmd := &cobra.Command{
		Use:   "delete-token",
		Short: "Deletes account token",
		Example: `# Delete token of the currently logged in account
athena account delete-token ID

# Delete token of the account with the specified name
athena account delete-token --account <account-name> ID`,
		Run: func(c *cobra.Command, args []string) {
			ctx := c.Context()

			if len(args) != 1 {
				c.HelpFunc()(c, args)
				os.Exit(1)
			}
			id := args[0]

			clientset := headless.NewClientOrDie(clientOpts, c)
			conn, client := clientset.NewAccountClientOrDie()
			defer utilio.Close(conn)
			if account == "" {
				account = getCurrentAccount(ctx, clientset).Username
			}
			promptUtil := utils.NewPrompt(clientOpts.PromptsEnabled)
			canDelete := promptUtil.Confirm(fmt.Sprintf("Are you sure you want to delete '%s' token? [y/n]", id))
			if canDelete {
				_, err := client.DeleteToken(ctx, &accountpkg.DeleteTokenRequest{Name: account, Id: id})
				errors.CheckError(err)
			} else {
				fmt.Printf("The command to delete '%s' was cancelled.\n", id)
			}
		},
	}
	cmd.Flags().StringVarP(&account, "account", "a", "", "Account name. Defaults to the current account.")
	return cmd
}
