package git

import (
	"git-backup-bot/config"
	"golang.org/x/oauth2"
	"context"
	"github.com/google/go-github/github"
	"gopkg.in/libgit2/git2go.v25"
	"regexp"
	"log"
	"os"
	"fmt"
	"errors"
)
// Main Object to Syncronize the Git.
type Synchronizer struct {
	config config.MainConfiguration
}

// Function than get the correct credentials for SSH
func (c Synchronizer) credentialsCallback(url string, username string, allowedTypes git.CredType) (git.ErrorCode, *git.Cred) {
	code, cred := git.NewCredSshKeyFromMemory("git", c.config.GitHub.PublicKey, c.config.GitHub.PrivateKey, c.config.GitHub.PassPhase)
	return git.ErrorCode(code), &cred
}

// If the git server don't have a good certificate we need overwrite the function to
// grant access
func certificateCheckCallback(_ *git.Certificate, _ bool, _ string) git.ErrorCode {
	return 0
}

// Constructor of the Syncronizer
func NewGitSyncronizer(config config.MainConfiguration) *Synchronizer {
	return &Synchronizer{
		config: config,
	}
}

// Main function of the syncronizer, loop of all repos registered and fetch the new elementes for the remotes
// Getting all the branches that exists and create
func (c *Synchronizer) Sync() {
	cbs := &git.RemoteCallbacks{
		CredentialsCallback:      c.credentialsCallback,
		CertificateCheckCallback: certificateCheckCallback,
	}
	ftOptions := &git.FetchOptions{
		RemoteCallbacks: *cbs,
	}
	cloneOptions := &git.CloneOptions{
		FetchOptions:ftOptions,
	}
	if _, err := os.Stat(c.config.WorkingFolder); os.IsNotExist(err) {
		log.Println("Initializing Working Directory")
		err = os.MkdirAll(c.config.WorkingFolder, os.ModePerm)
		if err != nil {
			log.Panic(err)
		}
	}
	for _, repoConfig := range c.getRepositories() {
		log.Println("Started Sync Repo: " + repoConfig.Name + " url: " + repoConfig.Url)
		folderRepo := c.getRepoFolder(repoConfig)
		repo := &git.Repository{}
		if _, err := os.Stat(folderRepo); os.IsNotExist(err) {
			log.Println("Repo not backup yet... ")
			repo, err = git.Clone(repoConfig.Url, folderRepo, cloneOptions)
			if err != nil {
				log.Panic(err)
			}
		} else {
			log.Println("Opening Repository... ")
			repo, _ = git.OpenRepository(folderRepo)
		}
		synchronizeBranchs(repo)
		branchIterator, _ := repo.NewBranchIterator(git.BranchRemote)
		branch, _, _ := branchIterator.Next()
		for branch != nil {
			branchRefName := branch.Reference.Name()
			localBranch := getLocalBranchWithRemoteBranch(*repo, *branch)
			if localBranch != nil { // In this moment the branchs will be solved
				log.Println("Ready for pull... with remote branch: " + branchRefName +
					" and local branch: " + localBranch.Reference.Name())
				commit, err := repo.LookupCommit(localBranch.Target())
				if err != nil {
					log.Panic(err)
				}
				err = repo.SetHead(localBranch.Reference.Name())
				if err != nil {
					log.Panic(err)
				}
				err = repo.ResetToCommit(commit, git.ResetHard, nil)
				if err != nil {
					log.Panic(err)
				}
				err = Pull(repo, ftOptions, branchRefName)
				if err != nil {
					log.Panic(err)
				}
			}

			branch, _, _ = branchIterator.Next()
		}

	}
}

// Get the folder using the working directory
func (c *Synchronizer) getRepoFolder(repo config.RepositoryConfiguration) string {
	if c.config.WorkingFolder[len(c.config.WorkingFolder) - 1] == '/' {
		return c.config.WorkingFolder + repo.Name
	} else {
		return c.config.WorkingFolder + "/" + repo.Name
	}
}

// Requesting to Github to get the repositories from Organization and appending the
// repos registered in the configuration
func (c *Synchronizer) getRepositories() []config.RepositoryConfiguration {
	repositories := []config.RepositoryConfiguration{}
	if len(c.config.Organization) > 0 {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: c.config.GitHub.AccessToken})
		tc := oauth2.NewClient(context.Background(), ts)
		client := github.NewClient(tc)
		repos, _, _ := client.Repositories.ListByOrg(c.config.Organization, nil)
		for _, repo := range repos {
			repoConfig := config.RepositoryConfiguration{
				Name: *repo.Name,
				Url:  *repo.SSHURL,
			}
			repositories = append(repositories, repoConfig)
		}

	}
	if len(c.config.Repositories) > 0 {
		for _, repo := range c.config.Repositories {
			repositories = append(repositories, repo)
		}
	}
	return repositories
}

// Do the pull because the libgit2 not do that automatic.
func Pull(repo *git.Repository, fetchOption *git.FetchOptions, branchRefName string) error {
	// Locate remote
	remote, err := repo.Remotes.Lookup("origin")
	if err != nil {
		return err
	}

	// Fetch changes from remote
	log.Println("Fetching Changes...")
	if err := remote.Fetch([]string{}, fetchOption, ""); err != nil {
		return err
	}

	// Get remote master
	remoteBranch, err := repo.References.Lookup(branchRefName)
	if err != nil {
		return err
	}

	remoteBranchID := remoteBranch.Target()
	// Get annotated commit
	annotatedCommit, err := repo.AnnotatedCommitFromRef(remoteBranch)
	if err != nil {
		return err
	}

	// Do the merge analysis
	mergeHeads := make([]*git.AnnotatedCommit, 1)
	mergeHeads[0] = annotatedCommit
	analysis, _, err := repo.MergeAnalysis(mergeHeads)
	if err != nil {
		return err
	}

	// Get repo head
	head, err := repo.Head()
	if err != nil {
		return err
	}
	if analysis&git.MergeAnalysisUpToDate != 0 {
		log.Println("Up To Date...")
		return nil

	} else if analysis&git.MergeAnalysisFastForward != 0 {
		log.Println("Merging Fast Forward...")
		// Fast-forward changes
		// Get remote tree

		commit, err := repo.LookupCommit(remoteBranchID)
		if err != nil {
			return err
		}

		remoteTree, err := commit.Tree()
		if err != nil {
			return err
		}

		// Checkout
		if err := repo.CheckoutTree(remoteTree, nil); err != nil {
			return err
		}

		repo.ResetToCommit(commit, git.ResetHard, nil)

	} else if analysis&git.MergeAnalysisNormal != 0 {
		log.Println("Merging With Commit...")
		// Just merge changes
		if err := repo.Merge([]*git.AnnotatedCommit{annotatedCommit}, nil, nil); err != nil {
			return err
		}
		// Check for conflicts
		index, err := repo.Index()
		if err != nil {
			return err
		}

		if index.HasConflicts() {
			return errors.New("Conflicts encountered. Please resolve them.")
		}

		// Make the merge commit
		sig, err := repo.DefaultSignature()
		if err != nil {
			return err
		}

		// Get Write Tree
		treeId, err := index.WriteTree()
		if err != nil {
			return err
		}

		tree, err := repo.LookupTree(treeId)
		if err != nil {
			return err
		}

		localCommit, err := repo.LookupCommit(head.Target())
		if err != nil {
			return err
		}

		remoteCommit, err := repo.LookupCommit(remoteBranchID)
		if err != nil {
			return err
		}

		repo.CreateCommit("HEAD", sig, sig, "", tree, localCommit, remoteCommit)

		// Clean up
		repo.StateCleanup()
	} else {
		return fmt.Errorf("Unexpected merge analysis result %d", analysis)
	}

	return nil
}

// Utils function that apply the regExp
func applyRegExp(a string) string {
	r, _ := regexp.Compile(`[a-z-A-Z_]+$`)
	return r.FindString(a)
}

// Getting the LocalBranch Using the refspec of the remote branch
func getLocalBranchWithRemoteBranch(repo git.Repository, remoteBranch git.Branch) (*git.Branch) {
	remoteBranchName, _ := remoteBranch.Name()
	remoteBranchConvertedName := applyRegExp(remoteBranchName)
	localBranches := extractBranches(repo, git.BranchLocal)
	for _, localBranch := range localBranches {
		localBranchName, _ := localBranch.Name()
		if localBranchName == remoteBranchConvertedName {
			return localBranch
		}
	}
	return nil
}

// Extract branches to array of branchs from iterator
func extractBranches(repo git.Repository, branchType git.BranchType) []*git.Branch {
	branches := []*git.Branch{}
	branchIterator, _ := repo.NewBranchIterator(branchType)
	branch, _, _ := branchIterator.Next()
	for branch != nil {
		branches = append(branches, branch)
		branch, _, _ = branchIterator.Next()
	}
	return branches
}

// Check if exists branch into branchs
func checkIfExists(branches []*git.Branch, branchName string) bool {
	for _, branch := range branches {
		branchArrName, _ := branch.Name()
		if branchArrName == branchName {
			return true
		}
	}
	return false
}

// Get all remotes branches and syncronize to the local repository
func synchronizeBranchs(repo *git.Repository) error {
	log.Println("Extract Local Branches")
	localBranches := extractBranches(*repo, git.BranchLocal)
	branchIterator, err := repo.NewBranchIterator(git.BranchRemote)
	if err != nil {
		return err
	}
	branch, _, err := branchIterator.Next()
	if err != nil {
		return err
	}
	for branch != nil {
		branchName, err := branch.Name()
		if err != nil {
			return err
		}
		r, _ := regexp.Compile(`[a-z]+$`)
		originName := r.FindString(branchName)
		if !checkIfExists(localBranches, originName) {
			log.Println("Branch " + originName + " not in the repository... Creating...")
			// Get the id of the last commit branch
			//branch.annotatedCommit, err := repo.AnnotatedCommitFromRef(branch.Reference)
			commit, err := repo.LookupCommit(branch.Target())
			if err != nil {
				return err
			}

			_, err = repo.CreateBranch(originName, commit, false)
			if err != nil {
				return err
			}
		}
		branch, _, _ = branchIterator.Next()
	}
	return nil
}

// Accomplish Cron Interface github.com/robfig/cron
func (c *Synchronizer) Run() {
	log.Println(" + Start Sync")
	c.Sync()
	log.Println(" - Finished Sync")
}
