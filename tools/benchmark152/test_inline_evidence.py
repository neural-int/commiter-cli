import importlib.util
import pathlib
import unittest

spec=importlib.util.spec_from_file_location('inline152',pathlib.Path(__file__).with_name('inline_evaluate.py'))
mod=importlib.util.module_from_spec(spec);spec.loader.exec_module(mod)


class SymbolEvidence(unittest.TestCase):
    def test_two_hop_maps_symbols_without_file_wide_merge(self):
        units=[dict(id='U1',file='F1',symbols=['First']),dict(id='U2',file='F1',symbols=['Second']),dict(id='U3',file='F2',symbols=['Check'])]
        graph={'edges':[{'from':'F2','caller':'Check','to':'R1','callee':'Bridge'}, {'from':'R1','caller':'Bridge','to':'F1','callee':'Second'}]}
        self.assertEqual(mod.relations(units,graph),[dict(from_unit='U3',to_unit='U2',hops=2,kind='soft_directed_call_path')])

    def test_common_callee_and_missing_graph_never_create_merge_edges(self):
        units=[dict(id='U1',file='F1',symbols=['A']),dict(id='U2',file='F2',symbols=['B'])]
        edges=[{'from':f,'caller':s,'to':'R1','callee':'Common'} for f,s in [('F1','A'),('F2','B')]]
        self.assertEqual(mod.relations(units,{'edges':edges}),[])
        self.assertEqual(mod.relations(units,{'edges':[]}),[])

    def test_relation_output_budget_fails_closed(self):
        units=[dict(id=f'U{i}',file='F1',symbols=['A']) for i in range(17)]+[dict(id=f'V{i}',file='F2',symbols=['B']) for i in range(17)]
        with self.assertRaisesRegex(ValueError,'relation_budget'):
            mod.relations(units,{'edges':[{'from':'F1','caller':'A','to':'F2','callee':'B'}]})


if __name__=='__main__': unittest.main()
