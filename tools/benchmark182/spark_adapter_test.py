import importlib.util,os,unittest,json
from pathlib import Path
AVAILABLE=importlib.util.find_spec('mlx') is not None and os.environ.get('COMMITER_SPARK_MODEL')
@unittest.skipUnless(AVAILABLE,'requires the approved Spark venv and explicit local model path; no inference')
class SparkBoundaryTests(unittest.TestCase):
    def setUp(self):
        import spark_adapter as a
        self.a=a
        self.vocab=a.LLTokenizer((Path(os.environ['COMMITER_SPARK_MODEL'])/'tokenizer.json').read_text(),n_vocab=131072,eos_token=1)
        self.schema={'type':'object','properties':{'answer':{'type':'string','enum':['ok']}},'required':['answer'],'additionalProperties':False}
    def test_thinking_free_then_json_mask(self):
        a=self.a;m=a.LLMatcher(self.vocab,a.LLMatcher.grammar_from_json_schema(self.schema),log_level=0);p=a.BoundaryProcessor(1,True,m);x=a.mx.zeros((1,131072));y=p(a.mx.array([3]),x);self.assertTrue(a.mx.all(y==0).item());y=p(a.mx.array([3,4]),x);self.assertFalse(p.thinking);self.assertEqual(y[0,1].item(),float('-inf'))
    def test_eos_before_close_is_error(self):
        a=self.a;p=a.BoundaryProcessor(1,True,None)
        with self.assertRaisesRegex(ValueError,'eos_before_thinking_close'):p(a.mx.array([3,1]),a.mx.zeros((1,131072)))
    def test_none_has_no_json_mask(self):
        a=self.a;p=a.BoundaryProcessor(1,False,None);x=a.mx.zeros((1,131072));self.assertTrue(a.mx.all(p(a.mx.array([4,100]),x)==0).item())
    def test_schema_accepts_only_expected_value(self):
        m=self.a.LLMatcher(self.vocab,self.a.LLMatcher.grammar_from_json_schema(self.schema),log_level=0)
        self.assertTrue(m.consume_tokens(self.vocab.tokenize_str('{"answer":"ok"}')));self.assertTrue(m.is_accepting())
if __name__=='__main__':unittest.main()
