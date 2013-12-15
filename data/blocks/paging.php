<?php 
if ($paging instanceof Paging):
        $total_page_number = $paging->getTotalPageNumber();
        $current_page = $paging->getCurrentPage();

        $start_page = min($current_page - $paging->CANDIDATE_PAGES_TO_BROWSE, 1);
        $end_page = max($current_page + $paging->CANDIDATE_PAGES_TO_BROWSE, $total_page_number);?>

        <?php if ($current_page != 1): ?>
                <button class="btn btn-default" data-toggle="button" 
                onclick="function(){location.href=<?=url("/app?id={$app->getId()}&page=".max($current_page - 1, 1))?>}">PREV
                </button>
        <?php endif; ?>
        
        <?php if ($start_page != 1): ?>
                ...
        <?php endif; ?>

        <?php for ($page = $start_page; $page <= $end_page; $page++): ?>
                <?php if ($page != $current_page): ?>
                        <button class="btn btn-default" data-toggle="button" 
                        onclick="function(){location.href=<?=url("/app?id={$app->getId()}&page=".$page)?>}"><?=$page?>
                        </button>
                <?php else: ?>
                        <button class="btn btn-default" data-toggle="button" >
                        <?=$page?>
                        </button>
                <?php endif; ?>
        <?php endfor; ?>
                        
        <?php if ($end_page != $total_page_number): ?>
                ...
        <?php endif; ?>

        <?php if (current_page != $total_page_number): ?>
                <button class="btn btn-default" data-toggle="button" 
                onclick="function(){location.href=<?=url("/app?id={$app->getId()}&page=".min($current_page + 1, $total_page_number))?>}">NEXT
                </button>
        <?php endif ?>
<?php endif ?>