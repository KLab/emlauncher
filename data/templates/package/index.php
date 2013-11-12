<div class="media">
  <p class="pull-left">
    <a href="<?=url("/app?id={$app->getId()}")?>">
      <img class="app-icon media-object img-rounded" src="<?=$app->getIconUrl()?>">
    </a>
  </p>
  <div class="media-body">
    <h2 class="media-hedding"><?=htmlspecialchars($app->getTitle())?></h2>
  </div>
</div>

<div class="row">
  <div class="col-sm-4 col-md-3 hidden-xs">
    <?=block('pkg_infopanel')?>
  </div>

  <div class="col-xs-12 col-sm-8 col-md-9">

    <div class="row">
      <div class="col-xs-7">
        <h3>
<?php if($package->getPlatform()=='Android'): ?>
          <i class="fa fa-android"></i>
<?php elseif($package->getPlatform()=='iOS'): ?>
          <i class="fa fa-apple"></i>
<?php else: ?>
          <i class="fa fa-question"></i>
<?php endif ?>
          <?=htmlspecialchars($package->getTitle())?>
        </h3>
        <p>
          <?=htmlspecialchars($package->getDescription())?>
        </p>
      </div>
      <div class="col-xs-5">
<!-- 
        <a href="<?=$package->getInstallUrl()?>" class="btn btn-primary col-xs-12"><i class="fa fa-download"></i> Install</a>
 -->
        <a href="<?=$package->getInstallUrl()?>" class="btn btn-success col-xs-12"><i class="fa fa-check"></i> Installed</a>
        <dl id="installed-date">
          <dt>Instaled at</dt>
          <dd>2013-12-15 13:12:05</dd>
        </dl>
      </div>
    </div>

    <p>
<?php foreach($package->getTags() as $tag): ?>
      <span class="label label-default"><?=htmlspecialchars($tag->getName())?></span>
<?php endforeach ?>
    </p>

    <dl class="dl-horizontal">
      <dt>Platform</dt>
<?php if($package->getPlatform()=='Android'): ?>
      <dd><i class="fa fa-android"></i> Android</dd>
<?php elseif($package->getPlatform()=='iOS'): ?>
      <dd><i class="fa fa-apple"></i> iOS</dd>
<?php else: ?>
      <dd><i class="fa fa-question"></i> unknown</dd>
<?php endif ?>
      <dt>Installed</dt>
      <dd>12<!-- fixme --></dd>
      <dt>Uploaded</dt>
      <dd><?=$package->getCreated()?></dd>
      <dt>Owners</dt>
<?php foreach($app->getOwners() as $owner):?>
        <dd><a href="mailto:<?=$owner->getOwnerMail()?>"><?=$owner->getOwnerMail()?></a></dd>
<?php endforeach ?>
    </dl>

  </div>
</div>

<div class="visible-xs">
  <?=block('pkg_infopanel')?>
</div>

