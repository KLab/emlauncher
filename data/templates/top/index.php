<div class="page-header row">
	<h2 class="headding col-xs-12 col-sm-8">EMLauncher
		<small class="subtitle">Only my APP can shoot it.</small>
	</h2>
	<div id="create-app-button" class="hidden-xs col-sm-4 text-right">
		<a class="btn btn-default" href="<?=url('/app/new')?>"><i class="fa fa-plus"></i> New Application</a>
	</div>
</div>

<div class="row">
<?php $counter = 0; ?>
<?php foreach($applications as $app): ?>
<div class="media app-list-item col-md-6">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon-sm media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h3 class="media-hedding">
      <a href="<?=url("/app?id={$app->getId()}")?>"><?=htmlspecialchars($app->getTitle())?></a>
<?php if(isset($comments[$app->getId()]) && $comments[$app->getId()]>0):?>
     <div class="balloon">
       <div title="<?=$comments[$app->getId()]?> comments"><?=$comments[$app->getId()]?></div>
     </div>
<?php endif?>
      <small title="Installed by <?=$app->getInstallUserCount()?> users" class="badge"><?=$app->getInstallUserCount()?></small>
    </h3>
    <p>
<?php
$upload_time = $app->getLastUpload();
$update_time = $upload_time?:$app->getCreated();
?>
      <?=($upload_time)?'last uploaded':'created'?>: <?=date('Y-m-d H:i',strtotime($update_time))?>
<?php if($login_user->getAppInstallDate($app) && $upload_time>$login_user->getAppInstallDate($app)): ?>
      <span class="label label-success">UPDATE</span>
<?php elseif(strtotime($update_time)>strtotime('yesterday')): ?>
      <span class="label label-primary">NEW</span>
<?php endif ?>
    </p>
  </div>
</div>
<?php if((++$counter)%2===0): ?>
</div>
<div class="row">
<?php endif ?>
<?php endforeach ?>
</div>

<div id="create-app-button" class="col-xs-12 visible-xs text-left">
  <a class="btn btn-default" href="<?=url('/app/new')?>"><i class="fa fa-plus"></i> New Application</a>
</div>

<div class="text-center">
  <?=block('paging',array('urlbase'=>url('/')))?>
</div>

<script type="text/javascript">

$('.app-list-item').on('click',function(event){
  $('a',this)[0].click();
});

</script>
