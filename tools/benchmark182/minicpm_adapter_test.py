import unittest
import mlx.core as mx
from minicpm_adapter import BoundaryState, Processor
class BoundaryTests(unittest.TestCase):
 def test_multiple_token_boundary_and_emitted_split(self):
  s=BoundaryState([7,8],{1,9});s.consume(7);s.consume(6);s.consume(7);self.assertIsNone(s.boundary);s.consume(8);self.assertEqual(s.boundary,2)
  self.assertEqual(s.split([7,6,7,8,10]),([7,6],[7,8],[10],True))
  self.assertEqual(s.split([7,6,7]),([7,6,7],[],[],False))
 def test_early_eos_is_failure(self):
  for t in [1,9]:
   with self.assertRaisesRegex(ValueError,'eos_before'):BoundaryState([7,8],{1,9}).consume(t)
 def test_free_thinking_no_grammar_and_boundary_consumed_once(self):
  s=BoundaryState([7,8],{1,9});p=Processor(2,s,None,32);logits=mx.zeros((32,))
  p(mx.array([30,31,4,7]),logits);self.assertEqual(s.tokens,[4,7]);p(mx.array([30,31,4,7,8]),logits);self.assertEqual(s.boundary,1)
  p(mx.array([30,31,4,7,8,11]),logits);self.assertEqual(s.tokens,[4,7,8,11])
if __name__=='__main__':unittest.main()
