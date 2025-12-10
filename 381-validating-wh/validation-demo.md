# MessageChannel - Create Tests

## Valid Slack MessageChannel

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: valid-slack-channel
  namespace: webhook-demo
spec:
  slack:
    channelID: C1234567890
  secretRef:
    name: valid-slack-secret
```

## Valid SMTP MessageChannel

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: valid-smtp-channel
  namespace: webhook-demo
spec:
  smtp:
    from: sender@example.com
    host: smtp.example.com
    port: 587
  secretRef:
    name: valid-smtp-secret
```

## No provider specified

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: no-provider
  namespace: webhook-demo
spec:
  secretRef:
    name: valid-slack-secret
```

## Multiple providers

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: multiple-providers
  namespace: webhook-demo
spec:
  slack:
    channelID: C2222222222
  smtp:
    from: test@example.com
    host: smtp.example.com
    port: 587
  secretRef:
    name: valid-slack-secret
```

## Missing secretRef name

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: no-secret-ref
  namespace: webhook-demo
spec:
  slack:
    channelID: C3333333333
  secretRef:
    name: ""
```

## Secret not found

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: missing-secret
  namespace: webhook-demo
spec:
  slack:
    channelID: C4444444444
  secretRef:
    name: does-not-exist
```

## Slack secret missing apiKey

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: invalid-slack-secret-keys
  namespace: webhook-demo
spec:
  slack:
    channelID: C5555555555
  secretRef:
    name: invalid-slack-secret
```

## SMTP secret missing password

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: incomplete-smtp-secret-keys
  namespace: webhook-demo
spec:
  smtp:
    from: sender@example.com
    host: smtp.example.com
    port: 587
  secretRef:
    name: incomplete-smtp-secret
```

## Missing Slack channelID

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: slack-no-channel-id
  namespace: webhook-demo
spec:
  slack:
    channelID: ""
  secretRef:
    name: valid-slack-secret
```

## SMTP missing from

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: smtp-no-from
  namespace: webhook-demo
spec:
  smtp:
    from: ""
    host: smtp.example.com
    port: 587
  secretRef:
    name: valid-smtp-secret
```

## SMTP missing host

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: smtp-no-host
  namespace: webhook-demo
spec:
  smtp:
    from: sender@example.com
    host: ""
    port: 587
  secretRef:
    name: valid-smtp-secret
```

## SMTP missing port

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: MessageChannel
metadata:
  name: smtp-no-port
  namespace: webhook-demo
spec:
  smtp:
    from: sender@example.com
    host: smtp.example.com
    port: 0
  secretRef:
    name: valid-smtp-secret
```

---

# MessageChannel - Update Tests

## Valid Slack update

```bash
kubectl patch messagechannel valid-slack-channel -n webhook-demo --type=merge -p '
{
  "spec": {
    "slack": { "channelID": "C9999999999" }
  }
}'
```

## Invalid update (adding second provider)

```bash
kubectl patch messagechannel valid-slack-channel -n webhook-demo --type=merge -p '
{
  "spec": {
    "smtp": {
      "from": "test@example.com",
      "host": "smtp.example.com",
      "port": 587
    }
  }
}'
```

---

# MessageChannel - Delete Test

```bash
kubectl delete messagechannel valid-smtp-channel -n webhook-demo
```

---

# ClusterMessageChannel - Create Tests

## Valid ClusterMessageChannel

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: ClusterMessageChannel
metadata:
  name: valid-cluster-channel
spec:
  slack:
    channelID: C8888888888
  secretRef:
    name: cluster-secret
    namespace: kargo-cluster-secrets
```

## Wrong namespace for secretRef

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: ClusterMessageChannel
metadata:
  name: cluster-channel-wrong-ns
spec:
  slack:
    channelID: C7777777777
  secretRef:
    name: cluster-secret
    namespace: wrong-namespace
```

---

# ClusterMessageChannel - Delete Test

```bash
kubectl delete clustermessagechannel valid-cluster-channel
```

---

# EventRouter - Create Tests

## Valid with MessageChannel

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: valid-router
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: MessageChannel
      name: valid-slack-channel
```

## Valid with ClusterMessageChannel

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: valid-router-cluster
  namespace: webhook-demo
spec:
  types:
    - PromotionFailed
  channels:
    - kind: ClusterMessageChannel
      name: valid-cluster-channel
```

## Empty channels list

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-no-channels
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels: []
```

## At least one valid channel

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-with-channels
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: MessageChannel
      name: valid-slack-channel
```

---

# EventRouter - Channel Validation Tests

## Invalid channel kind

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-invalid-kind
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: TotallyInvalidKind
      name: some-channel
```

## Empty channel kind

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-empty-kind
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: ""
      name: some-channel
```

## Empty channel name

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-empty-name
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: MessageChannel
      name: ""
```

## ClusterMessageChannel should not have namespace

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-cluster-with-ns
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: ClusterMessageChannel
      name: valid-cluster-channel
      namespace: some-namespace
```

## MessageChannel wrong namespace

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-wrong-ns
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: MessageChannel
      name: valid-slack-channel
      namespace: different-namespace
```

## MessageChannel correct namespace

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-matching-ns
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: MessageChannel
      name: valid-slack-channel
      namespace: webhook-demo
```

## MessageChannel without namespace (should default)

```yaml
apiVersion: ee.kargo.akuity.io/v1alpha1
kind: EventRouter
metadata:
  name: router-default-ns
  namespace: webhook-demo
spec:
  types:
    - PromotionSucceeded
  channels:
    - kind: MessageChannel
      name: valid-slack-channel
```

---

# EventRouter - Update Tests

## Valid update (add channels)

```bash
kubectl patch eventrouter valid-router -n webhook-demo --type=merge -p '
{
  "spec": {
    "channels": [
      { "kind": "MessageChannel", "name": "valid-slack-channel" },
      { "kind": "ClusterMessageChannel", "name": "valid-cluster-channel" }
    ]
  }
}'
```

## Invalid update (invalid kind)

```bash
kubectl patch eventrouter valid-router -n webhook-demo --type=merge -p '
{
  "spec": {
    "channels": [
      { "kind": "InvalidKind", "name": "whatever" }
    ]
  }
}'
```

---

# EventRouter - Delete Test

```bash
kubectl delete eventrouter router-matching-ns -n webhook-demo
```
