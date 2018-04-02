# conju
Event/invitation/email management system

## Development Environment Setup

Set up Github account (https://github.com/)  

Add ssh keys to Github account ([Instructions](https://help.github.com/enterprise/2.12/user/articles/generating-a-new-ssh-key-and-adding-it-to-the-ssh-agent/))  
(Optional: also add ssh key to ~/.ssh/authorized_keys on minimancer for ease of ssh/scp)

Install Google Cloud SDK	https://cloud.google.com/sdk/  
Initialize Cloud SDK ([Instructions](https://cloud.google.com/sdk/docs/initializing))  
Choose project "shabscott" for dev environment. (prod environment to follow)  

Includes followup info about login support (google accounts) and datastore usage in Go.

```
$ eval $(ssh-agent -s)
$ ssh-add
$ cd  ~/work/conju
$ git init
$ git remote add origin git@github.com:cshabsin/conju.git                                                                                              
$ git pull origin master
```

Install Go tools (apt-get install golang-go on Debian, not sure what this is on Mac).  
Set GOPATH environment variable somewhere useful (like ~/gopath).  
Run "go get" to fetch dependencies into GOPATH.

Install the gofmt git pre-commit hook to be warned when you attempt to
commit a change with non-gofmt'ed changes.

```
$ curl -o .git/hooks/pre-commit https://golang.org/misc/git/pre-commit
$ chmod a+x .git/hooks/pre-commit
```

Run "dev_appserver.py app.yaml" to test on localhost:8080. Admin console at localhost:8000.

### Emacs go mode setup

(Only seems to work with Emacs 24)

```
$ mkdir -p ~/.emacs.d/lisp
$ cd ~/.emacs.d/lisp
$ curl -O https://raw.githubusercontent.com/dominikh/go-mode.el/master/go-mode.el
```

Add to ~/.emacs.d/init.el:
```lisp
(add-to-list 'load-path "/home/cshabsin/.emacs.d/lisp")
(require 'go-mode)
```

## Prod stuff

Make sure to get real_import_data into place from the Drive
folder. Also replace placeholder email addresses hard-coded in source
before deploying (look for `****` in at least `email.go` and
`invitations.go`).

To deploy to AppEngine, use

```
$ gcloud app deploy
```

(Add `--project project-id` if needed.)

Make sure you also deploy the datastore indexes:

```
$ gcloud datastore create-indexes index.yaml
```

## Useful Links

Useful Go tutorial: http://tour.golang.org/  
Useful codelab for "hello world" in Go: https://cloud.google.com/appengine/docs/standard/go/quickstart
