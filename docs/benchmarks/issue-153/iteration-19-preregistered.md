## 固定同範囲比較

iteration16/18の先頭4file/2intentを固定C全段で計測。profile/prompt/model/予算/byte guard/8unit cap変更なし、最大6call retry0。使用済みprojectionでholdoutではない。atom品質を保存し、各fileが一unitならfile pair FM/FSも記録してbaselineと同尺度で比較。baselineはfinal metadata/Validateまで、Cはpartitionまでで成果物範囲が異なるためplan valid/コストを同等に扱わない。片方の品質エラーを重み無しで優劣へ換算しない。
