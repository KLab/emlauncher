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
          <?=block('platform_icon')?>
          <?=htmlspecialchars($package->getTitle())?>
        </h3>
        <p>
          <?=htmlspecialchars($package->getDescription())?>
        </p>
      </div>
      <div class="col-xs-5">
<?php if($login_user->getPackageInstalledDate($package)): ?>
        <a href="<?=$package->getInstallUrl()?>" class="btn btn-success col-xs-12"><i class="fa fa-check"></i> Installed</a>
        <dl id="installed-date">
          <dt>Instaled at</dt>
          <dd><?=$login_user->getPackageInstalledDate($package)?></dd>
        </dl>
<?php else: ?>
        <a href="<?=$package->getInstallUrl()?>" class="btn btn-primary col-xs-12"><i class="fa fa-download"></i> Install</a>
<?php endif ?>
      </div>
    </div>

    <p>
<?php foreach($package->getTags() as $tag): ?>
      <span class="label label-default"><?=htmlspecialchars($tag->getName())?></span>
<?php endforeach ?>
    </p>

    <dl class="dl-horizontal">
      <dt>Platform</dt>
      <dd><?=block('platform_icon',array('with_name'=>true))?></dd>
      <dt>Installed</dt>
      <dd><?=$package->getInstallCount()?></dd>
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

