# tea

<p align="center">
<img width="200" src="/teapartybiglogo.png" alt="Material Bread logo">
<p align="center">
The application interface for `TeaParty` 
</p>
</p>


## Design / Overview

`TeaParty` consists of two core parts `tea` & `party`, as illustrated by the diagram below. 

* `party` - A suite of services and logic that support the platform. This is where all the magic happens 🪄 ( If this sounds complicated, just think of `party` as the 🧠 "brains" 🧠 of the project .) 

* `tea` - A suite of services and logic to make it easy for users to interact with `party` 

<p align="center">
<img width="200" src="/teadiagram.png" alt="Material Bread logo">
<p align="center">


`tea` was elected to be designed as a desktop application over a hosted web service in order to provide users with the most control, safety, privacy, and security, while interacting with the marketplace. (**Note** Although `tea` is packaged as a desktop application, it does not have to run on your local desktop! In fact it is quite happy living on a remote server.)

`tea` works by first checking for a local [NKN](https://nkn.org/) wallet, `wallet`, file and begins listing to this address. If this file does not exist, a new account is created for the user. (**NOTE** This file, `wallet`, is very imporant and should be treated as any other wallet or private key. Do not delete, move, or alter this file while you have open or pending trades as your NKN public address is how `party` talks to your `tea` client.)

After starting the NKN connection, `tea` also begins serving the static assets located at `kodata` (our React application) while exposing several API endpoints for the user to interact with `Party` 

`tea` is now at your disposal to interact with 🎉`Party`🎉

### Current Supported Assets

`TeaParty` currently supports the trade of the following assets:
* Ethereum
* Polygon
* Celo
* Solana


### Asset Support Comming Soon

I am currently in the process of introducing the following assets into `TeaParty`

* NFT's (on all supported chains) 
* Kaspa
* Radiant
* Bitcoin
* Raven
* Ergo
* .... Want `TeaParty` to support something not on the roadmap? [submit](https://github.com/TeaPartyCrypto/Tea/issues) an issue and let me know! 


## Getting Started
**NOTE** Tea is currently in BETA and the only server avalible is the staging enviorment. In the staging environment there are several **IMPORTANT** differences from the production environment:

1. All of the RPC's are pointing to the following networks 

(**DO NOT SEND MAINNET CURRENCY**)

       * Ethereum: Goerli ([faucet](https://www.alchemy.com/overviews/goerli-faucet))

       * Polygon: Mumbai ([faucet](https://faucet.polygon.technology/))

       * Solana: Testnet ([faucet](https://solfaucet.com/))

       * Celo: Alfajores ([faucet](https://celo.org/developers/faucet))

1. The watch timeout has been taken down to 300 secconds from 2 hours ( After 300 secconds any pending transaction will fail) 

1. Currently, users still have to pay for the transaction fees ( I will personally reimburse any and all MO costs incurred while playing on testnet untill the faucet is setup. 


### System Prerequisites for Running `tea`

If you need help getting started, feel free to join us in the MineOnlium [Discord!](https://discord.gg/4JFjejV4FN) or PM me directly on Discord @ Filth#5858 (439229993625714688)

You can also post in the Github [Discussions](https://github.com/TeaPartyCrypto/Tea/discussions)

Here are a few things you can currently do with `tea`:

* Browse the marketplace. 
* Create new orders. 
* View acquired Private Keys.
* Remove acquired Private Keys from the local filesystem. 
* Interact with the `TeaParty` smart contract to pay for transaction fees, currently set @ 1 MO/ Transaction. ( **NOTE:** ALL MO Accounts with a balance over 17k have FREE access to Buy and Sell with `TeaParty`) 


