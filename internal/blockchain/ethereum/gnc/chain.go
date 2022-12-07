package gnc

const chainYAML = `
accounts:
  - name: leader
    mnemonic: "apology false junior asset sphere puppy upset dirt miracle rice horn spell ring vast wrist crisp snake oak give cement pause swallow barely clever"
    coin_type: 118
    algo: sm2
    # address: gnc1qxcrws5mytpzwkk4tn4dyysw2ru96na3kvhk65
    coins: ["99999000000ugnc"]
    roles: ["ROOT_ADMIN"]
  - name: validator1
    mnemonic: "gossip wheel net riot retreat arrest ozone dragon funny undo bulb visa victory label slim domain network wage suit peanut tattoo text venture answer"
    coin_type: 118
    algo: sm2
    # address: gnc13d59wpwv6swsn8z5xwk4vr2n67q20sps9zzefd
    coins: ["10000000ugnc"]
{VAR_ACCOUNTS}
validators:
  - name: validator1
    sef_delegation: "10000000ugnc"
    commission_rate:
    commission_max_rate:
    commission_max_change_rate:
    min_self_delegation:
    moniker:
    identity:
    website:
    security_contact:
    details:
init:
  config:
    p2p:
      addr_book_strict: false
    rpc:
      timeout_broadcast_tx_commit: 60s
    consensus:
      timeout_commit: {VAR_BLOCK_PERIOD}
  app:
    api:
      enabled-unsafe-cors: true
    json-rpc:
      api: "eth,net,web3,txpool,debug,personal"
    minimum-gas-prices: "0ugnc"
  client:
genesis:
  genesis_time: 2022-04-12T05:35:29Z
  chain_id: {VAR_CHAINID}
  app_state:
    auth:
      params:
        max_memo_characters: "20971520"
    permission:
      params:
        enabled: false
    evm:
      params:
        evm_denom: "ugnc"
    feemarket:
      params:
        no_base_fee: true
    bank:
      denom_metadata:
        - description: "base denom of gnc block chain"
          base: "ugnc"
          display: "gnc"
          denom_units:
            - denom: ugnc
              exponent: 0
            - denom: gnc
              exponent: 6
          name: "gnc network"
          symbol: "GNC"
    crisis:
      constant_fee:
        denom: ugnc
    staking:
      params:
        bond_denom: ugnc
        unbonding_time: 10s
    gov:
      deposit_params:
        min_deposit:
          - denom: ugnc
            amount: "10000000"
    mint:
      minter:
        inflation: "0.000000000000000000"
      params:
        mint_denom: ugnc
        inflation_rate_change: "0.000000000000000000"
        inflation_min: "0.000000000000000000"
`
