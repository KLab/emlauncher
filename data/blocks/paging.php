<?php
$candidate = 5;
if($paging && $paging instanceof Paging):
$cur_page = $paging->getCurrentPage();
$max_page = $paging->getTotalPageNumber();

if($max_page <= $candidate+4){ // 省略不要
	$start = 1;
	$end = $max_page;
}
elseif($cur_page <= ceil($candidate/2)+1){ // 後半のみ省略
	$start = 1;
	$end = $candidate + 2;
}
elseif($cur_page >= $max_page-floor($candidate/2)-2){ // 前半のみ省略
	$end = $max_page;
	$start = $end - $candidate - 1;
}
else{ // 前後省略
	$start = max(1,$cur_page-ceil($candidate/2)+1);
	$end = min($max_page,$start+$candidate-1);
}
?>
<ul class="pagination">
	<li<?=($cur_page==1)?' class="disabled"':''?>>
		<a href="<?=($cur_page==1)?'#':mfwHttp::composeUrl($urlbase,array('page'=>$cur_page-1))?>">&laquo;</a>
	</li>
<?php if($start!=1): ?>
	<li<?=($cur_page==1)?' class="active"':''?>>
		<a href="<?=mfwHttp::composeUrl($urlbase,array('page'=>1))?>">1</a>
	</li>
<?php endif ?>
<?php if($start>2): ?>
	<li class="disabled">
		<a href="#">...</a>
	</li>
<?php endif ?>
<?php for($i=$start;$i<=$end;++$i): ?>
	<li<?=($i==$cur_page)?' class="active"':''?>>
		<a href="<?=mfwHttp::composeUrl($urlbase,array('page'=>$i))?>"><?=$i?></a>
	</li>
<?php endfor ?>
<?php if($end<$max_page-1): ?>
	<li class="disabled">
		<a href="#">...</a>
	</li>
<?php endif ?>
<?php if($end!=$max_page): ?>
	<li<?=($cur_page==$max_page)?' class="active"':''?>>
		<a href="<?=mfwHttp::composeUrl($urlbase,array('page'=>$max_page))?>"><?=$max_page?></a>
	</li>
<?php endif ?>
	<li<?=($cur_page==$max_page)?' class="disabled"':''?>>
		<a href="<?=($cur_page==$max_page)?'#':mfwHttp::composeUrl($urlbase,array('page'=>$cur_page+1))?>">&raquo;</a>
	</li>
</ul>
<?php endif ?>
