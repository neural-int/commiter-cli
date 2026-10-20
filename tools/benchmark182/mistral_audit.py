"""Verify the official tokenizer against the pinned MLX byte vocabulary.

This command performs no model inference. The two Instruct reserved-slot name
differences are recorded, not hidden; official input serialization is used.
"""
import argparse
import hashlib
import json
from pathlib import Path

from mistral_common.tokens.tokenizers.base import SpecialTokenPolicy
from mistral_common.tokens.tokenizers.mistral import MistralTokenizer

from mistral_tokenize import tokenize


def audit(assets, model, requests, reasoning):
    tokenizer = MistralTokenizer.from_file(str(assets / 'tekken.json')).instruct_tokenizer.tokenizer
    converted = json.loads((model / 'tokenizer.json').read_text())
    # GPT-2 byte-to-Unicode mapping used by the converted tokenizer.
    byte_values = list(range(33, 127)) + list(range(161, 173)) + list(range(174, 256))
    characters = byte_values[:]
    offset = 0
    for byte in range(256):
        if byte not in byte_values:
            byte_values.append(byte)
            characters.append(256 + offset)
            offset += 1
    reverse = {chr(character): byte for byte, character in zip(byte_values, characters)}
    added = {entry['id']: entry['content'] for entry in converted['added_tokens']}
    mismatches = []
    for text, token_id in converted['model']['vocab'].items():
        actual = added[token_id].encode() if token_id in added else bytes(reverse[c] for c in text)
        expected = tokenizer.id_to_byte_piece(token_id, SpecialTokenPolicy.KEEP)
        if actual != expected:
            mismatches.append({'id': token_id, 'converted': actual.decode(), 'official': expected.decode()})
    expected_mismatches = [] if reasoning else [34, 35]
    assert sorted(entry['id'] for entry in mismatches) == expected_mismatches, mismatches
    rows = []
    for request_path in sorted(requests.glob('*request.json')):
        if 'original' in request_path.name:
            continue
        serialized = tokenize(json.loads(request_path.read_text()), assets, reasoning)
        assert serialized['tokens'][0] == tokenizer.bos_id
        assert serialized['tokens'][-1] == 4
        assert tokenizer.eos_id not in serialized['tokens']
        rows.append({'request': request_path.name, 'tokens': serialized['tokens'],
                     'sha256': hashlib.sha256(json.dumps(serialized['tokens']).encode()).hexdigest()})
    assert len(rows) == 3
    left, right = rows[1]['tokens'], rows[2]['tokens']
    prefix = next((i for i, pair in enumerate(zip(left, right)) if pair[0] != pair[1]), min(len(left), len(right)))
    assert 4 < prefix < min(len(left), len(right))
    return {'vocab_size': len(converted['model']['vocab']), 'mismatches': mismatches,
            'eos': tokenizer.eos_id, 'prefix_tokens': prefix, 'requests': rows}


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    for name in ['assets', 'model', 'requests', 'out']:
        parser.add_argument('--' + name, type=Path, required=True)
    parser.add_argument('--reasoning', action='store_true')
    args = parser.parse_args()
    args.out.write_text(json.dumps(audit(args.assets, args.model, args.requests, args.reasoning), indent=2) + '\n')
