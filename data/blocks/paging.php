<?php
$candidate = 5;

if($max_page <= $candidate+2){ // 省略不要
	$start = 1;
	$end = $max_page;
}
elseif($cur_page <= ceil($candidate/2)){ // 後半のみ省略
	$start = 1;
	$end = $candidate + 1;
}
elseif($cur_page >= $max_page-floor($candidate/2)){ // 前半のみ省略
	$end = $max_page;
	$start = $end - $candidate;
}
else{ // 前後省略
	$start = max(1,$cur_page-ceil($candidate/2)+1);
	$end = min($max_page,$start+$candidate-1);
}
?>
<ul class="pagination  pagination-sm">
<?php if($start!=1): ?>
	<li<?=($cur_page==1)?' class="active"':''?>>
		<a href="<?=mfwHttp::composeUrl($urlbase,array('page'=>1))?>">1</a>
	</li>
<?php endif ?>
<?php if($start>2): ?>
	<li class="disabled omission">
		<span>..</span>
	</li>
<?php endif ?>
<?php for($i=$start;$i<=$end;++$i): ?>
	<?php if($i==$cur_page): ?>
		<li class="disabled active"><span><?=$i?></span></li>
	<?php else: ?>
		<li><a href="<?=mfwHttp::composeUrl($urlbase,array('page'=>$i))?>"><?=$i?></a></li>
	<?php endif ?>
<?php endfor ?>
<?php if($end<$max_page-1): ?>
	<li class="disabled omission">
		<span>..</span>
	</li>
<?php endif ?>
<?php if($end!=$max_page): ?>
	<li<?=($cur_page==$max_page)?' class="active"':''?>>
		<a href="<?=mfwHttp::composeUrl($urlbase,array('page'=>$max_page))?>"><?=$max_page?></a>
	</li>
<?php endif ?>
</ul>

