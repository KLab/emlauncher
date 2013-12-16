<?php 
$candidate_page_number = 5;
if ($paging instanceof Paging):
	$total_page_number = $paging->getTotalPageNumber();
	$current_page = $paging->getCurrentPage();

	$start_page = max($current_page - $candidate_page_number, 1);
	$end_page = min($current_page + $candidate_page_number, $total_page_number);?>
	<ul class="pagination">
		<li><a href='<?=url("/top?page=1")?>'><span>&laquo;&laquo;</span></a></li>
		<li><a href='<?=url("/top?page=".max($current_page - 1, 1))?>'><span>&laquo;</span></a></li>
		<?php if ($start_page != 1): ?>
			<li>...</li>
		<?php endif; ?>

		<?php for ($page = $start_page; $page <= $end_page; $page++): ?>
			<?php if ($page != $current_page): ?>
				<li><a href='<?=url("/top?page=".$page)?>'><span><?=$page?></span></a></li>
			<?php else: ?>
				<li class="active"><a href='#'><?=$page?><span class="sr-only">(current)</span></a></li>
			<?php endif; ?>
		<?php endfor; ?>

		<?php if ($end_page != $total_page_number): ?>
			<li>...</li>
		<?php endif; ?>

		<li><a href='<?=url("/top?page=".min($current_page + 1, $total_page_number))?>'><span>&raquo;</span></a></li>
		<li><a href='<?=url("/top?page=".$total_page_number)?>'><span>&raquo;&raquo;</span></a></li>
	</ul>
<?php endif ?>