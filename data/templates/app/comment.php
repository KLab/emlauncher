<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><a href="<?=url("/app?id={$app->getId()}")?>"><?=htmlspecialchars($app->getTitle())?></a></h2>
    <p><?=nl2br(htmlspecialchars($app->getDescription()))?></p>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('app_infopanel')?>
  </div>

  <div class="col-xs-12 col-sm-8 col-md-9">

	<div id="comment_form">
	</div>

	<div id="comments">
	  <h3><?=$comment_count?> comments</h3>
      <ul class="list-group">
<?php
foreach($comments as $c):
    $pkg = ($c->getPackageId())? $commented_package[$c->getPackageId()]: null;
	$comment_page = floor(($comment_count-$c->getNumber())/$comments_in_page)+1;
?>
        <li class="list-group-item" id="comment-<?=$c->getNumber()?>">
		  <dl class="dl-horizontal">
			<dt><a href="<?=url("/app/comment?id={$app->getId()}&page=$comment_page#comment-{$c->getNumber()}")?>"><?=$c->getNumber()?></a></dt>
			<dd><?=htmlspecialchars($c->getMessage())?></dd>
		  </dl>
		  <div class="text-right">
<?php if($pkg): ?>
			<a href="<?=url("/package?id={$pkg->getId()}")?>">
			  <?=block('platform_icon',array('package'=>$pkg))?> <?=htmlspecialchars($pkg->getTitle())?></a>
<?php else: ?>
			<span>No package installed</span>
<?php endif ?>
			(<?=$c->getCreated('Y-m-d H:i')?>)
		  </div>
        </li>
<?php endforeach ?>
      </ul>
    </div>

	<div class="text-center">
	  <?=block('paging',array('urlbase'=>mfwRequest::url()))?>
	</div>

  </div>
</div>

<div class="visible-xs">
  <?=block('app_infopanel')?>
</div>

<script type="text/javascript">

</script>
