# Secret Sync Operator

Currently, an update to a Secret will not cause deployments that are currently using it to redeploy. This causes the deployments to still retain the old config until it redeploys due to another reason. This operator will watch changes to a secret and redeploy any deployments that have subscribed to it if it changes.

## Installation
Install the operator with
`kubectl apply -f deploy`

## Running it locally
Pull the repo and run it with `operator-sdk up local` while connected to your kubernetes cluster.

## Usage
Secrets and Deployments that subscribe to those secrets should be labelled with "sso.gable.dev/secret:${NAME_OF_SECRET}"

A demonstration is located in the examples folder

## Contributing
Pull requests are welcome. For major changes, please open an issue first to discuss what you would like to change.

Please make sure to update tests as appropriate.

## License
[MIT](https://choosealicense.com/licenses/mit/)