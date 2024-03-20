from web3 import Web3
import time
import web3
import random
import sys
from hexbytes import HexBytes

poolmgr_url = 'http://localhost:8545'
ganache_url = 'http://localhost:8467'

poolmgr = Web3(Web3.HTTPProvider(poolmgr_url))
web3 = Web3(Web3.HTTPProvider(ganache_url))

account_1 = '0x67b1d87101671b127f5f8714789C7192f7ad340e'
private_key1 = '0x26e86e45f6fc45ec6e2ecd128cec80fa1d1505e5507dcd2ae58c3130a7a97b48'

txs_to_send = []

nonce = web3.eth.get_transaction_count(account_1)
# nonce = 0

balance = web3.eth.get_balance(account_1)

def generate_ethereum_address():
    return '0x' + ''.join(random.choices('0123456789abcdef', k=40))

for index, _ in enumerate(range(1)):
    tx = {
        'chainId': 999999,
        'nonce': nonce + index,
        'to': HexBytes(generate_ethereum_address()),
        'value': web3.to_wei(0.000000001, 'ether'),
        'gas': 2000000,
        'gasPrice': web3.to_wei(1, 'gwei')
    }

    txs_to_send.append(tx)

#txs_to_send.reverse()

random.shuffle(txs_to_send)

signed_txs = []

print(f"Signing txs")

for index, tx in enumerate(txs_to_send):
    sys.stdout.write(f"{index} {tx['nonce']}\n")
    sys.stdout.flush()  # Ensure it gets displayed
    signed_tx = web3.eth.account.sign_transaction(tx, private_key1)
    signed_txs.append(signed_tx)

hashes_for_receipts = []

print(f"Sending txs")

start_time = time.time()

for index, signed_tx in enumerate(signed_txs):
    sys.stdout.write(f"\r{index}")
    sys.stdout.flush()  # Ensure it gets displayed
    try:
        tx_hash = poolmgr.eth.send_raw_transaction(signed_tx.rawTransaction)
        hashes_for_receipts.append(tx_hash)
    except Exception as e:
        print(f"Error sending tx {index}, error: {repr(e)}")
        pass

print(f"Sent txs")


# enough to wait for the last receipt
# tx_hash = hashes_for_receipts[-1]
# web3.eth.wait_for_transaction_receipt(tx_hash)

for tx_hash in hashes_for_receipts:
    hex_hash = web3.to_hex(tx_hash)
    print("waiting for receipt tx_hash " + str(hex_hash))
    web3.eth.wait_for_transaction_receipt(tx_hash)
    print("tx receipt received for tx_hash " + str(hex_hash))

end_time = time.time()

width = 30

tps = len(hashes_for_receipts) / (end_time - start_time)

print("duration", end_time - start_time)

print(f"{str(balance).rjust(width)},{str(nonce).rjust(width)},{str(tps).rjust(width)}")