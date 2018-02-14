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

$ eval $(ssh-agent -s)
$ ssh-add
$ cd  ~/work/conju
$ git init
$ git remote add origin git@github.com:cshabsin/conju.git                                                                                              
$ git pull origin master

Install Go tools (apt-get install golang-go on Debian, not sure what this is on Mac).
Set GOPATH environment variable somewhere useful (like ~/gopath).
Run "go get" to fetch dependencies into GOPATH.

Run "dev_appserver.py app.yaml" to test on localhost:8080. Admin console at localhost:8000.

### Emacs go mode setup

(Only seems to work with Emacs 24)

$ mkdir -p ~/.emacs.d/lisp
$ cd ~/.emacs.d/lisp
$ wget https://raw.githubusercontent.com/dominikh/go-mode.el/master/go-mode.el

Add to ~/.emacs.d/init.el:
(add-to-list 'load-path "/home/cshabsin/.emacs.d/lisp")
(require 'go-mode)

## Useful Links

Useful Go tutorial: http://tour.golang.org/ 
Useful codelab for "hello world" in Go: https://cloud.google.com/appengine/docs/standard/go/quickstart
