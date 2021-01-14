# gitlab-reviewdog-webhook

Trigger [reviewdog](https://github.com/reviewdog/reviewdog) checks on a repository via [GitLab webhook](https://docs.gitlab.com/ee/user/project/integrations/webhooks.html) rather than CI job.

## Usage

1. Clone:

    ```bash
    git clone https://github.com/orsinium-labs/reviewdog-gitlab-webhook.git
    cd reviewdog-gitlab-webhook
    go get github.com/rakyll/statik
    ```

1. Make config:

    ```bash
    cp config_example.toml config.toml
    nano config.toml
    ```

1. Build: `./build.sh`
1. Run: `./review.bin`
1. Add webhook into GitLab project: `http://my_server:8080/review?secret=something-random`
