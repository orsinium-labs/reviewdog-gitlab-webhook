# gitlab-reviewdog-webhook

Trigger [reviewdog](https://github.com/reviewdog/reviewdog) checks on a repository via GitLab webhook rather than CI job.

```bash
go get github.com/rakyll/statik
git clone https://github.com/orsinium-labs/reviewdog-gitlab-webhook.git
cd reviewdog-gitlab-webhook
cp config_example.toml config.toml
nano config.toml
./build.sh
```

It produces `review.bin` binary with the config already inside.
