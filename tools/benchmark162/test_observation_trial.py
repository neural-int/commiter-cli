import unittest
from observation_trial import choose

class ObservationSafety(unittest.TestCase):
    def observations(self, a=True, b=True):
        return {name:{'pass':value,'tests_run':1} for name,value in [('before',True),('only_a',a),('only_b',b),('both',True),('after',True)]}

    def test_pass_with_zero_or_nonzero_test_count_does_not_certify_independence(self):
        for count in (0,1,100):
            observed=self.observations()
            for state in observed.values():state['tests_run']=count
            self.assertEqual(choose(observed,[],True)[0],'defer')
            self.assertEqual(choose(observed,[],False)[0],'keep_separate')

    def test_invalid_endpoint_rejects_apparently_directional_partial_result(self):
        for endpoint in ('before','both','after'):
            observed=self.observations(False,True)
            observed[endpoint]['pass']=False
            self.assertEqual(choose(observed,[],True)[0],'defer')

    def test_partial_failure_order_is_not_reversed(self):
        for a,b,provider,consumer in [(True,False,'EA','EB'),(False,True,'EB','EA')]:
            decision,edges,_=choose(self.observations(a,b),[],True)
            self.assertEqual(decision,'keep_separate')
            self.assertEqual((edges[-1]['provider'],edges[-1]['consumer']),(provider,consumer))

if __name__=='__main__':unittest.main()
