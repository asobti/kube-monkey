# Notifications

kube-monkey can notify an endpoint of your choice after an attack. It can be a Slack webhook
or a custom API.

## Settings

| Setting | Type | Default | Description |
| --- | --- | --- | --- |
| `notifications.enabled` | bool | `false` | Report attacks to an HTTP endpoint |
| `notifications.reportSchedule` | bool | `false` | Report the schedule as well as the attacks |
| `notifications.proxy` | string | none | Proxy URL to send the requests through |
| `notifications.attacks` | receiver | empty | The endpoint, message and headers attacks are posted to |

## Example

```toml
[notifications]
  enabled = true
  reportSchedule = true
  [notifications.attacks]
    endpoint = "http://url1"
    message = "message1"
    headers = ["header1Key:header1Value","header2Key:header2/Value"]
```

## Message placeholders

The message supports the following placeholders:

| Placeholder | Value |
| --- | --- |
| `{$name}` | Victim's name |
| `{$kind}` | Victim's kind |
| `{$namespace}` | Victim's namespace |
| `{$timestamp}` | Attack's time from the Unix epoch, in milliseconds |
| `{$time}` | Attack's time |
| `{$date}` | Attack's date |
| `{$error}` | Result's error, if any |
| `{$kubemonkeyid}` | kube-monkey id, set with the `KUBE_MONKEY_ID` environment variable, otherwise empty |

```toml
message = '{
          "what": "Kube-monkey(${kubemonkeyid}) attack of {$name} in {$namespace}",
          "who": "{$name}",
          "when": {$timestamp}
         }'
```

## Header placeholders

Headers support a special placeholder that reads an environment variable. This is useful when
calling an API with a protected endpoint. The typical case is an API token passed to the
kube-monkey container from a Kubernetes Secret.

```toml
headers = ["api-key:{$env:API_TOKEN}", "Content-Type:application/json"]
```

`{$env:API_TOKEN}` is replaced by the value of the `API_TOKEN` environment variable.

!!! note "A missing variable does not stop the notification"

    If the environment variable does not exist, the notification call is **not** cancelled.
    The value resolves to an empty string and a warning shows up in the logs.

## Example configmap

A ready-made example lives at
[`examples/notifications-configmap.yaml`](https://github.com/asobti/kube-monkey/tree/master/examples/notifications-configmap.yaml).

## With the Helm chart

```bash
helm install my-release kubemonkey/kube-monkey \
  --set config.notifications.enabled=true \
  --set config.notifications.endpoint=http://localhost:8080/path \
  --set config.notifications.message="{\"foo\":\"bar\"}" \
  --set config.notifications.headers="Content-Type:application/json\"\,\"client-id:kubemonkey"
```
